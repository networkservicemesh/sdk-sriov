// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !windows
// +build !windows

package cgroup

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

const (
	mkdirPerm = 0o750
)

// NewFakeCgroup creates and returns a new cgroup for testing with some k8s default devices allowed.
func NewFakeCgroup(ctx context.Context, path string) (*Cgroup, error) {
	cg, deviceSupplier, err := newFakeCgroup(ctx, path)
	if err != nil {
		return nil, err
	}

	if err = deviceSupplier("c 136:* rwm"); err != nil {
		return nil, err
	}
	if err = deviceSupplier("c *:* m"); err != nil {
		return nil, err
	}
	if err = deviceSupplier("b *:* m"); err != nil {
		return nil, err
	}

	return cg, err
}

// NewFakeWideCgroup creates and returns a new cgroup for testing with "a *:* rwm" allowed
func NewFakeWideCgroup(ctx context.Context, path string) (*Cgroup, error) {
	cg, deviceSupplier, err := newFakeCgroup(ctx, path)
	if err != nil {
		return nil, err
	}

	if err = deviceSupplier("a *:* rwm"); err != nil {
		return nil, err
	}

	return cg, nil
}

func newFakeCgroup(ctx context.Context, path string) (*Cgroup, supplierFunc, error) {
	if err := os.MkdirAll(path, mkdirPerm); err != nil {
		return nil, nil, err
	}
	go func() {
		<-ctx.Done()
		_ = os.RemoveAll(path)
	}()

	var lock sync.Mutex
	cgStub := new(fakeCgroupDevices)

	deviceSupplier := outputFileAPI(filepath.Join(path, deviceListFileName))
	if err := deviceSupplier(""); err != nil {
		return nil, nil, err
	}

	if err := inputFileAPI(ctx, filepath.Join(path, deviceAllowFileName), func(s string) {
		lock.Lock()
		defer lock.Unlock()

		dev, _ := parseDevice(s)
		cgStub.allow(dev)

		_ = deviceSupplier(cgStub.String())
	}); err != nil {
		return nil, nil, err
	}

	if err := inputFileAPI(ctx, filepath.Join(path, deviceDenyFileName), func(s string) {
		lock.Lock()
		defer lock.Unlock()

		dev, _ := parseDevice(s)
		cgStub.deny(dev)

		_ = deviceSupplier(cgStub.String())
	}); err != nil {
		return nil, nil, err
	}

	cgroups, err := NewCgroups(path)
	if err != nil {
		return nil, nil, err
	}
	if len(cgroups) != 1 {
		return nil, nil, errors.Errorf("expected exactly 1 cgroup for path: %s", path)
	}

	return cgroups[0], deviceSupplier, nil
}

type fakeCgroupDevices struct {
	devices []*device
}

func (s *fakeCgroupDevices) String() string {
	sb := new(strings.Builder)
	for _, dev := range s.devices {
		sb.WriteString(dev.String())
	}
	return sb.String()
}

func (s *fakeCgroupDevices) allow(dev *device) {
	for i := 0; i < len(s.devices); {
		d := s.devices[i]

		if reflect.DeepEqual(d, dev) || d.isWiderThan(dev) {
			return
		}

		if dev.isWiderThan(d) {
			s.devices = append(s.devices[:i], s.devices[i+1:]...)
			continue
		}

		i++
	}
	s.devices = append(s.devices, dev)
}

func (s *fakeCgroupDevices) deny(dev *device) {
	for i := 0; i < len(s.devices); {
		d := s.devices[i]

		if reflect.DeepEqual(d, dev) || dev.isWiderThan(d) {
			s.devices = append(s.devices[:i], s.devices[i+1:]...)
			continue
		}

		if d.isWiderThan(dev) {
			for mode := range dev.Modes {
				delete(d.Modes, mode)
			}
			if len(d.Modes) == 0 {
				s.devices = append(s.devices[:i], s.devices[i+1:]...)
				continue
			}
		}

		i++
	}
}
