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

//go:build !windows && perm
// +build !windows,perm

package vfio_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
)

const (
	serverSocket = "server.socket"
	cgroupDir    = "cgroup_dir"
)

func testServer(ctx context.Context, tmpDir string) (*grpc.ClientConn, error) {
	socketURL := &url.URL{
		Scheme: "unix",
		Path:   filepath.Join(tmpDir, serverSocket),
	}

	server := grpc.NewServer()
	networkservice.RegisterNetworkServiceServer(server, mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
		vfiomech.MECHANISM: &vfioForwarderStub{
			iommuGroup:  iommuGroup,
			vfioMajor:   1,
			vfioMinor:   2,
			deviceMajor: 3,
			deviceMinor: 4,
		},
	}))
	_ = grpcutils.ListenAndServe(ctx, socketURL, server)

	<-time.After(1 * time.Millisecond) // wait for the server to start

	return grpc.DialContext(ctx, socketURL.String(), grpc.WithInsecure())
}

func TestVFIOClient_RequestPerm(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	tmpDir := filepath.Join(os.TempDir(), t.Name())
	err := os.MkdirAll(tmpDir, 0o750)
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cc, err := testServer(ctx, tmpDir)
	require.NoError(t, err)
	defer func() { _ = cc.Close() }()

	client := chain.NewNetworkServiceClient(
		vfio.NewClient(vfio.WithVFIODir(tmpDir), vfio.WithCgroupDir(cgroupDir)),
		networkservice.NewNetworkServiceClient(cc),
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	})
	require.NoError(t, err)

	mech := vfiomech.ToMechanism(conn.GetMechanism())
	require.NotNil(t, mech)
	require.Equal(t, cgroupDir, mech.GetCgroupDir())

	info := new(unix.Stat_t)

	err = unix.Stat(filepath.Join(tmpDir, vfioDevice), info)
	require.NoError(t, err)
	require.Equal(t, uint32(1), vfio.Major(info.Rdev))
	require.Equal(t, uint32(2), vfio.Minor(info.Rdev))

	err = unix.Stat(filepath.Join(tmpDir, iommuGroupString), info)
	require.NoError(t, err)
	require.Equal(t, uint32(3), vfio.Major(info.Rdev))
	require.Equal(t, uint32(4), vfio.Minor(info.Rdev))

	require.NoError(t, ctx.Err())
}

type vfioForwarderStub struct {
	iommuGroup  uint
	vfioMajor   uint32
	vfioMinor   uint32
	deviceMajor uint32
	deviceMinor uint32
}

func (vf *vfioForwarderStub) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mech := vfiomech.ToMechanism(request.GetConnection().GetMechanism()); mech != nil {
		mech.SetIommuGroup(vf.iommuGroup)
		mech.SetVfioMajor(vf.vfioMajor)
		mech.SetVfioMinor(vf.vfioMinor)
		mech.SetDeviceMajor(vf.deviceMajor)
		mech.SetDeviceMinor(vf.deviceMinor)
	}

	return next.Server(ctx).Request(ctx, request)
}

func (vf *vfioForwarderStub) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
