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

// Package vfio provides server, vfioClient chain elements for the VFIO mechanism connection
package vfio

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-sriov/pkg/tools/cgroup"
)

type vfioServer struct {
	vfioDir        string
	cgroupBaseDir  string
	deviceCounters map[string]int
	lock           sync.Mutex
}

// NewServer returns a new VFIO server chain element
func NewServer(vfioDir, cgroupBaseDir string) networkservice.NetworkServiceServer {
	return &vfioServer{
		vfioDir:        vfioDir,
		cgroupBaseDir:  cgroupBaseDir,
		deviceCounters: map[string]int{},
	}
}

func (s *vfioServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("vfioServer", "Request")

	if mech := vfio.ToMechanism(request.GetConnection().GetMechanism()); mech != nil {
		if mech.GetCgroupDir() == "" {
			return nil, errors.New("expected client cgroup directory set")
		}

		vfioMajor, vfioMinor, err := s.getDeviceNumbers(filepath.Join(s.vfioDir, vfioDevice))
		if err != nil {
			logger.Errorf("failed to get device numbers for the device: %v", vfioDevice)
			return nil, err
		}

		igid := mech.GetParameters()[vfio.IommuGroupKey]
		deviceMajor, deviceMinor, err := s.getDeviceNumbers(filepath.Join(s.vfioDir, igid))
		if err != nil {
			logger.Errorf("failed to get device numbers for the device: %v", igid)
			return nil, err
		}

		cgroupDirPattern := filepath.Join(s.cgroupBaseDir, mech.GetCgroupDir())

		if err := func() error {
			s.lock.Lock()
			defer s.lock.Unlock()

			if err := s.deviceAllow(cgroupDirPattern, vfioMajor, vfioMinor); err != nil {
				logger.Errorf("failed to allow device for the client: %v", vfioDevice)
				return err
			}
			mech.SetVfioMajor(vfioMajor)
			mech.SetVfioMinor(vfioMinor)

			if err := s.deviceAllow(cgroupDirPattern, deviceMajor, deviceMinor); err != nil {
				logger.Errorf("failed to allow device for the client: %v", igid)
				_ = s.deviceDeny(cgroupDirPattern, vfioMajor, vfioMinor)
				return err
			}
			mech.SetDeviceMajor(deviceMajor)
			mech.SetDeviceMinor(deviceMinor)

			return nil
		}(); err != nil {
			return nil, err
		}
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		s.close(ctx, request.GetConnection())
		return nil, err
	}

	return conn, nil
}

func (s *vfioServer) getDeviceNumbers(deviceFile string) (major, minor uint32, err error) {
	info := new(unix.Stat_t)
	if err := unix.Stat(deviceFile, info); err != nil {
		return 0, 0, err
	}
	return Major(info.Rdev), Minor(info.Rdev), nil
}

func (s *vfioServer) deviceAllow(cgroupDirPattern string, major, minor uint32) error {
	cgroups, err := cgroup.NewCgroups(cgroupDirPattern)
	if err != nil || len(cgroups) == 0 {
		return errors.Wrapf(err, "no cgroupDir found: %s", cgroupDirPattern)
	}

	for _, cg := range cgroups {
		isWider, err := cg.IsWiderThan(major, minor)
		if err != nil {
			return err
		}
		if isWider {
			continue
		}

		key := deviceKey(cg.Path, major, minor)
		if counter, ok := s.deviceCounters[key]; ok && counter > 0 {
			s.deviceCounters[key] = counter + 1
			return nil
		}

		if err := cg.Allow(major, minor); err != nil {
			return err
		}

		s.deviceCounters[key] = 1
	}

	return nil
}

func (s *vfioServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.close(ctx, conn)

	if _, err := next.Server(ctx).Close(ctx, conn); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *vfioServer) close(ctx context.Context, conn *networkservice.Connection) {
	logger := log.FromContext(ctx).WithField("vfioServer", "close")

	if mech := vfio.ToMechanism(conn.GetMechanism()); mech != nil {
		cgroupDirPattern := filepath.Join(s.cgroupBaseDir, mech.GetCgroupDir())

		s.lock.Lock()
		defer s.lock.Unlock()

		vfioMajor := mech.GetVfioMajor()
		vfioMinor := mech.GetVfioMinor()
		if !(vfioMajor == 0 && vfioMinor == 0) {
			if err := s.deviceDeny(cgroupDirPattern, vfioMajor, vfioMinor); err != nil {
				logger.Warnf("failed to deny device for the client: %v", vfioDevice)
			}
		}

		deviceMajor := mech.GetDeviceMajor()
		deviceMinor := mech.GetDeviceMinor()
		if !(deviceMajor == 0 && deviceMinor == 0) {
			if err := s.deviceDeny(cgroupDirPattern, deviceMajor, deviceMinor); err != nil {
				logger.Warnf("failed to deny device for the client: %v", mech.GetIommuGroup())
			}
		}
	}
}

func (s *vfioServer) deviceDeny(cgroupDirPattern string, major, minor uint32) error {
	cgroups, err := cgroup.NewCgroups(cgroupDirPattern)
	if err != nil || len(cgroups) == 0 {
		return errors.Wrapf(err, "no cgroupDir found: %s", cgroupDirPattern)
	}

	for _, cg := range cgroups {
		isWider, err := cg.IsWiderThan(major, minor)
		if err != nil {
			return err
		}
		if isWider {
			continue
		}

		key := deviceKey(cg.Path, major, minor)
		s.deviceCounters[key]--
		if s.deviceCounters[key] > 0 {
			return nil
		}

		if err := cg.Deny(major, minor); err != nil {
			return err
		}
	}

	return nil
}

func deviceKey(cgroupDir string, major, minor uint32) string {
	return fmt.Sprintf("%s:%d:%d", cgroupDir, major, minor)
}
