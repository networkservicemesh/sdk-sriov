// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type device struct {
	Type  string
	Major string
	Minor string
	Modes map[rune]struct{}
}

func newDevice(major, minor uint32, modes ...rune) *device {
	d := &device{
		Type:  "c",
		Major: strconv.FormatUint(uint64(major), 10),
		Minor: strconv.FormatUint(uint64(minor), 10),
		Modes: make(map[rune]struct{}),
	}

	for _, mode := range modes {
		d.Modes[mode] = struct{}{}
	}

	return d
}

var devicePattern = regexp.MustCompile("(?P<type>[abc]) (?P<major>[*0-9]+):(?P<minor>[*0-9]+) (?P<mode>[rwm]+)")

func parseDevice(s string) (*device, error) {
	if !devicePattern.MatchString(s) {
		return nil, errors.Errorf("invalid device string: %s", s)
	}

	split := devicePattern.FindAllStringSubmatch(s, -1)[0]

	d := &device{
		Type:  split[devicePattern.SubexpIndex("type")],
		Major: split[devicePattern.SubexpIndex("major")],
		Minor: split[devicePattern.SubexpIndex("minor")],
		Modes: make(map[rune]struct{}),
	}

	for _, mode := range split[devicePattern.SubexpIndex("mode")] {
		d.Modes[mode] = struct{}{}
	}

	return d, nil
}

func (d *device) String() string {
	sb := new(strings.Builder)

	sb.WriteString(d.Type)
	sb.WriteString(" ")

	sb.WriteString(d.Major)
	sb.WriteString(":")
	sb.WriteString(d.Minor)
	sb.WriteString(" ")

	for mode := range d.Modes {
		sb.WriteRune(mode)
	}
	sb.WriteRune('\n')

	return sb.String()
}

func (d *device) isWiderThan(other *device) (isWider bool) {
	switch d.Type {
	case other.Type:
	case "a":
		isWider = true
	default:
		return false
	}

	switch {
	case d.Major == other.Major:
	case d.Major == "*":
		isWider = true
	default:
		return false
	}

	switch {
	case d.Minor == other.Minor:
	case d.Minor == "*":
		isWider = true
	default:
		return false
	}

	for mode := range other.Modes {
		if _, ok := d.Modes[mode]; !ok {
			return false
		}
	}
	if len(d.Modes) > len(other.Modes) {
		isWider = true
	}

	return isWider
}
