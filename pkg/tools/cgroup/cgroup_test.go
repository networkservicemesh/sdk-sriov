// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

//+build !windows

package cgroup_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk-sriov/pkg/tools/cgroup"
)

const (
	mkdirPerm          = 0750
	deviceListFileName = "devices.list"
)

func createCgroup(t *testing.T, path string) {
	require.NoError(t, os.MkdirAll(path, mkdirPerm))

	f, err := os.Create(filepath.Join(path, deviceListFileName))
	require.NoError(t, err)
	_ = f.Close()
}

func TestNewCgroups(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	createCgroup(t, filepath.Join(tmpDir, "a"))
	createCgroup(t, filepath.Join(tmpDir, "b"))
	createCgroup(t, filepath.Join(tmpDir, "c"))

	cgroups, err := cgroup.NewCgroups(filepath.Join(tmpDir, "*"))
	require.NoError(t, err)
	require.Len(t, cgroups, 3)

	require.Equal(t, filepath.Join(tmpDir, "a"), cgroups[0].Path)
	require.Equal(t, filepath.Join(tmpDir, "b"), cgroups[1].Path)
	require.Equal(t, filepath.Join(tmpDir, "c"), cgroups[2].Path)
}

func TestCgroup_IsWiderThan(t *testing.T) {
	samples := []struct {
		name   string
		device string
		result bool
	}{
		{
			name:   "All major",
			device: "a *:2 rwm",
			result: true,
		},
		{
			name:   "All minor",
			device: "a 1:* rwm",
			result: true,
		},
		{
			name:   "Char major",
			device: "c *:2 rwm",
			result: true,
		},
		{
			name:   "Char minor",
			device: "c 1:* rwm",
			result: true,
		},
		{
			name:   "All",
			device: "a 1:2 rwm",
			result: true,
		},
		{
			name:   "Char",
			device: "c 1:2 rwm",
			result: false,
		},
		{
			name:   "Block",
			device: "b *:* rwm",
			result: false,
		},
	}

	tmpDir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	createCgroup(t, tmpDir)

	cgroups, err := cgroup.NewCgroups(tmpDir)
	require.NoError(t, err)

	cg := cgroups[0]

	for i := range samples {
		sample := samples[i]
		t.Run(sample.name, func(t *testing.T) {
			err := ioutil.WriteFile(filepath.Join(tmpDir, deviceListFileName), []byte(sample.device), 0)
			require.NoError(t, err)

			isWider, err := cg.IsWiderThan(1, 2)
			require.NoError(t, err)
			require.Equal(t, sample.result, isWider)
		})
	}
}
