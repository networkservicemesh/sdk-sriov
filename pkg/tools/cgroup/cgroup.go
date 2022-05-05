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
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
)

const (
	deviceListFileName  = "devices.list"
	deviceAllowFileName = "devices.allow"
	deviceDenyFileName  = "devices.deny"
)

// Cgroup represents linux devices cgroup
type Cgroup struct {
	Path string
}

// NewCgroups returns all cgroups matching pathPattern
func NewCgroups(pathPattern string) (cgroups []*Cgroup, err error) {
	var filePaths []string
	filePaths, err = filepath.Glob(filepath.Join(pathPattern, deviceListFileName))
	if err != nil {
		return nil, err
	}

	for _, filePath := range filePaths {
		cgroups = append(cgroups, &Cgroup{Path: filepath.Dir(filePath)})
	}

	return cgroups, nil
}

// Allow allows "c major:minor rwm" for cgroup
func (c *Cgroup) Allow(major, minor uint32) error {
	dev := newDevice(major, minor, 'r', 'w', 'm')

	return ioutil.WriteFile(filepath.Join(c.Path, deviceAllowFileName), []byte(dev.String()), 0)
}

// Deny denies "c major:minor rw" for cgroup
func (c *Cgroup) Deny(major, minor uint32) error {
	dev := newDevice(major, minor, 'r', 'w')

	return ioutil.WriteFile(filepath.Join(c.Path, deviceDenyFileName), []byte(dev.String()), 0)
}

// IsAllowed returns if "c major:minor rwm" is allowed for cgroup
func (c *Cgroup) IsAllowed(major, minor uint32) (bool, error) {
	isAllowed, _, err := c.compareTo(newDevice(major, minor, 'r', 'w', 'm'))
	return isAllowed, err
}

// IsWiderThan returns if cgroup allows wider device group than "c major:minor rwm":
//   - "a *:minor rwm"
//   - "a major:* rwm"
//   - "a *:* rwm"
//   - "c *:minor rwm"
//   - "c major:* rwm"
//   - "c *:* rwm"
func (c *Cgroup) IsWiderThan(major, minor uint32) (bool, error) {
	_, isWider, err := c.compareTo(newDevice(major, minor, 'r', 'w', 'm'))
	return isWider, err
}

func (c *Cgroup) compareTo(dev *device) (isAllowed, isWider bool, err error) {
	file, err := os.Open(filepath.Join(c.Path, deviceListFileName))
	if err != nil {
		return false, false, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		d, err := parseDevice(scanner.Text())
		if err != nil {
			return false, false, err
		}

		if reflect.DeepEqual(d, dev) {
			isAllowed = true
		} else if d.isWiderThan(dev) {
			return true, true, nil
		}
	}
	return isAllowed, false, nil
}
