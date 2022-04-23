// Copyright (c) 2022 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package vfio_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/cgroup"
)

const (
	testWait = 100 * time.Millisecond
	testTick = testWait / 100
)

func eventuallyIsAllowed(t *testing.T, a1 *cgroup.Cgroup, major, minor uint32) bool {
	finishTime := time.Now().Add(testWait)
	for time.Now().Before(finishTime) {
		allowed, allowedErr := a1.IsAllowed(major, minor)
		require.NoError(t, allowedErr)

		if allowed {
			return true
		}
		time.Sleep(testTick)
	}
	return false
}

func testCgroups(ctx context.Context, t *testing.T, tmpDir string) (notAllowed, allowed, wider *cgroup.Cgroup) {
	var err error

	notAllowed, err = cgroup.NewFakeCgroup(ctx, filepath.Join(tmpDir, uuid.NewString()))
	require.NoError(t, err)

	allowed, err = cgroup.NewFakeCgroup(ctx, filepath.Join(tmpDir, uuid.NewString()))
	require.NoError(t, err)

	require.NoError(t, allowed.Allow(1, 2))
	require.NoError(t, allowed.Allow(3, 4))

	require.True(t, eventuallyIsAllowed(t, allowed, 1, 2))
	require.True(t, eventuallyIsAllowed(t, allowed, 3, 4))

	wider, err = cgroup.NewFakeWideCgroup(ctx, filepath.Join(tmpDir, uuid.NewString()))
	require.NoError(t, err)

	require.True(t, eventuallyIsAllowed(t, wider, 1, 2))
	require.True(t, eventuallyIsAllowed(t, wider, 3, 4))

	return notAllowed, allowed, wider
}

func TestVFIOServer_Request(t *testing.T) {
	t.Skip("https://github.com/networkservicemesh/sdk-sriov/issues/336")

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	tmpDir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	server := chain.NewNetworkServiceServer(
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			vfiomech.MECHANISM: vfio.NewServer(tmpDir, tmpDir),
		}),
	)

	notAllowed, allowed, wider := testCgroups(ctx, t, tmpDir)

	err := unix.Mknod(filepath.Join(tmpDir, vfioDevice), unix.S_IFCHR|0o666, int(unix.Mkdev(1, 2)))
	require.NoError(t, err)
	err = unix.Mknod(filepath.Join(tmpDir, iommuGroupString), unix.S_IFCHR|0o666, int(unix.Mkdev(3, 4)))
	require.NoError(t, err)

	conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: vfiomech.MECHANISM,
				Parameters: map[string]string{
					vfiomech.CgroupDirKey:  "*",
					vfiomech.IommuGroupKey: iommuGroupString,
				},
			},
		},
	})
	require.NoError(t, err)

	mech := vfiomech.ToMechanism(conn.GetMechanism())
	require.NotNil(t, mech)
	require.Equal(t, uint32(1), mech.GetVfioMajor())
	require.Equal(t, uint32(2), mech.GetVfioMinor())
	require.Equal(t, uint32(3), mech.GetDeviceMajor())
	require.Equal(t, uint32(4), mech.GetDeviceMinor())

	// Wait for the fileapi hooks
	time.Sleep(testWait)

	allowed12, err := allowed.IsAllowed(1, 2)
	require.NoError(t, err)
	require.True(t, allowed12)

	allowed34, err := allowed.IsAllowed(3, 4)
	require.NoError(t, err)
	require.True(t, allowed34)

	wider12, err := wider.IsAllowed(1, 2)
	require.NoError(t, err)
	require.True(t, wider12)

	wider34, err := wider.IsAllowed(3, 4)
	require.NoError(t, err)
	require.True(t, wider34)

	notAllowed12, err := notAllowed.IsAllowed(1, 2)
	require.NoError(t, err)
	require.True(t, notAllowed12)

	notAllowed34, err := notAllowed.IsAllowed(3, 4)
	require.NoError(t, err)
	require.True(t, notAllowed34)

	require.NoError(t, ctx.Err())
}

func TestVFIOServer_Close(t *testing.T) {
	t.Skip("https://github.com/networkservicemesh/sdk-sriov/issues/336")

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	tmpDir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	server := chain.NewNetworkServiceServer(
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			vfiomech.MECHANISM: vfio.NewServer(tmpDir, tmpDir),
		}),
	)

	notAllowed, allowed, wider := testCgroups(ctx, t, tmpDir)

	conn := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Cls:  cls.LOCAL,
			Type: vfiomech.MECHANISM,
			Parameters: map[string]string{
				vfiomech.CgroupDirKey:   "*",
				vfiomech.IommuGroupKey:  iommuGroupString,
				vfiomech.VfioMajorKey:   "1",
				vfiomech.VfioMinorKey:   "2",
				vfiomech.DeviceMajorKey: "3",
				vfiomech.DeviceMinorKey: "4",
			},
		},
	}

	_, err := server.Close(ctx, conn)
	require.NoError(t, err)

	require.False(t, eventuallyIsAllowed(t, notAllowed, 1, 2))
	require.False(t, eventuallyIsAllowed(t, notAllowed, 3, 4))

	wider12, err := wider.IsAllowed(1, 2)
	require.NoError(t, err)
	require.True(t, wider12)

	wider34, err := wider.IsAllowed(3, 4)
	require.NoError(t, err)
	require.True(t, wider34)

	allowed12, err := allowed.IsAllowed(1, 2)
	require.NoError(t, err)
	require.False(t, allowed12)

	allowed34, err := allowed.IsAllowed(3, 4)
	require.NoError(t, err)
	require.False(t, allowed34)

	require.NoError(t, ctx.Err())
}
