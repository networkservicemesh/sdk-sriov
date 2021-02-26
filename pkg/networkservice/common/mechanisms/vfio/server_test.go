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

//+build !windows

package vfio_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/cgroup"
)

func testCgroups(ctx context.Context, t *testing.T, tmpDir string) (notAllowed, allowed, wider *cgroup.Cgroup) {
	var err error

	notAllowed, err = cgroup.NewFakeCgroup(ctx, filepath.Join(tmpDir, uuid.NewString()))
	require.NoError(t, err)

	allowed, err = cgroup.NewFakeCgroup(ctx, filepath.Join(tmpDir, uuid.NewString()))
	require.NoError(t, err)

	require.NoError(t, allowed.Allow(1, 2))
	require.NoError(t, allowed.Allow(3, 4))

	wider, err = cgroup.NewFakeWideCgroup(ctx, filepath.Join(tmpDir, uuid.NewString()))
	require.NoError(t, err)

	return notAllowed, allowed, wider
}

func TestVFIOServer_Request(t *testing.T) {
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

	err := unix.Mknod(filepath.Join(tmpDir, vfioDevice), unix.S_IFCHR|0666, int(unix.Mkdev(1, 2)))
	require.NoError(t, err)
	err = unix.Mknod(filepath.Join(tmpDir, iommuGroupString), unix.S_IFCHR|0666, int(unix.Mkdev(3, 4)))
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

	require.Never(t, func() bool {
		allowed12, err := allowed.IsAllowed(1, 2)
		require.NoError(t, err)

		allowed34, err := allowed.IsAllowed(3, 4)
		require.NoError(t, err)

		return !allowed12 || !allowed34
	}, 100*time.Millisecond, 10*time.Millisecond)

	require.Never(t, func() bool {
		wider12, err := wider.IsAllowed(1, 2)
		require.NoError(t, err)

		wider34, err := wider.IsAllowed(3, 4)
		require.NoError(t, err)

		return !wider12 || !wider34
	}, 100*time.Millisecond, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		_12, err := notAllowed.IsAllowed(1, 2)
		require.NoError(t, err)

		_34, err := notAllowed.IsAllowed(3, 4)
		require.NoError(t, err)

		return _12 && _34
	}, 100*time.Millisecond, 10*time.Millisecond)

	require.NoError(t, ctx.Err())
}

func TestVFIOServer_Close(t *testing.T) {
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

	require.Never(t, func() bool {
		notAllowed12, err := notAllowed.IsAllowed(1, 2)
		require.NoError(t, err)

		notAllowed34, err := notAllowed.IsAllowed(3, 4)
		require.NoError(t, err)

		return notAllowed12 || notAllowed34
	}, 100*time.Millisecond, 10*time.Millisecond)

	require.Never(t, func() bool {
		wider12, err := wider.IsAllowed(1, 2)
		require.NoError(t, err)

		wider34, err := wider.IsAllowed(3, 4)
		require.NoError(t, err)

		return !wider12 || !wider34
	}, 100*time.Millisecond, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		allowed12, err := allowed.IsAllowed(1, 2)
		require.NoError(t, err)

		allowed34, err := allowed.IsAllowed(3, 4)
		require.NoError(t, err)

		return !allowed12 && !allowed34
	}, 100*time.Millisecond, 10*time.Millisecond)

	require.NoError(t, ctx.Err())
}
