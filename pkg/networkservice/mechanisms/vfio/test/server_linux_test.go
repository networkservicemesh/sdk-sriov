// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package vfio_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vfioapi "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/sriovtest"
)

const (
	deviceAllowFile    = "devices.allow"
	deviceDenyFile     = "devices.deny"
	deviceStringFormat = "c %d:%d rwm"
)

func testConnection() *networkservice.Connection {
	return &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Cls:  cls.LOCAL,
			Type: vfioapi.MECHANISM,
		},
		Context: &networkservice.ConnectionContext{
			ExtraContext: map[string]string{
				clientCgroupDirKey: "",
			},
		},
	}
}

func testVfioServer(ctx context.Context, t *testing.T, allowedDevices *allowedDevices) (server networkservice.NetworkServiceServer, tmpDir string) {
	tmpDir = path.Join(os.TempDir(), t.Name())
	err := os.MkdirAll(tmpDir, 0750)
	assert.Nil(t, err)

	server = chain.NewNetworkServiceServer(
		vfio.NewServer(tmpDir, tmpDir),
		&endpointStub{
			igid: iommuGroup,
		},
	)

	err = sriovtest.InputFileAPI(ctx, path.Join(tmpDir, deviceAllowFile), func(s string) {
		var major, minor int
		_, _ = fmt.Sscanf(s, deviceStringFormat, &major, &minor)
		allowedDevices.Lock()
		allowedDevices.devices[fmt.Sprintf("%d:%d", major, minor)] = true
		allowedDevices.Unlock()
	})
	assert.Nil(t, err)
	err = sriovtest.InputFileAPI(ctx, path.Join(tmpDir, deviceDenyFile), func(s string) {
		var major, minor int
		_, _ = fmt.Sscanf(s, deviceStringFormat, &major, &minor)
		allowedDevices.Lock()
		delete(allowedDevices.devices, fmt.Sprintf("%d:%d", major, minor))
		allowedDevices.Unlock()
	})
	assert.Nil(t, err)

	return server, tmpDir
}

func TestVfioServer_Request(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	allowedDevices := &allowedDevices{
		devices: map[string]bool{},
	}
	server, tmpDir := testVfioServer(ctx, t, allowedDevices)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	err := unix.Mknod(path.Join(tmpDir, vfioDevice), unix.S_IFCHR|0666, int(unix.Mkdev(1, 2)))
	assert.Nil(t, err)
	err = unix.Mknod(path.Join(tmpDir, iommuGroup), unix.S_IFCHR|0666, int(unix.Mkdev(3, 4)))
	assert.Nil(t, err)

	conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: testConnection(),
	})
	assert.Nil(t, err)

	assert.Equal(t, "1", conn.Mechanism.Parameters[vfioMajorKey])
	assert.Equal(t, "2", conn.Mechanism.Parameters[vfioMinorKey])
	assert.Equal(t, "3", conn.Mechanism.Parameters[deviceMajorKey])
	assert.Equal(t, "4", conn.Mechanism.Parameters[deviceMinorKey])

	assert.Eventually(t, func() bool {
		allowedDevices.Lock()
		defer allowedDevices.Unlock()
		return reflect.DeepEqual(map[string]bool{
			"1:2": true,
			"3:4": true,
		}, allowedDevices.devices)
	}, time.Second, 10*time.Millisecond)
}

func TestVfioServer_Close(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	allowedDevices := &allowedDevices{
		devices: map[string]bool{
			"1:2": true,
			"3:4": true,
		},
	}
	server, tmpDir := testVfioServer(ctx, t, allowedDevices)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	conn := testConnection()
	conn.Mechanism.Parameters = map[string]string{
		vfio.IommuGroupKey: iommuGroup,
		vfioMajorKey:       "1",
		vfioMinorKey:       "2",
		deviceMajorKey:     "3",
		deviceMinorKey:     "4",
	}

	_, err := server.Close(ctx, conn)
	assert.Nil(t, err)

	assert.Eventually(t, func() bool {
		allowedDevices.Lock()
		defer allowedDevices.Unlock()
		return reflect.DeepEqual(map[string]bool{}, allowedDevices.devices)
	}, time.Second, 10*time.Millisecond)
}

type endpointStub struct {
	igid string
}

func (e *endpointStub) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Mechanism.Parameters = map[string]string{
		vfio.IommuGroupKey: e.igid,
	}
	return next.Server(ctx).Request(ctx, request)
}

func (e *endpointStub) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

type allowedDevices struct {
	devices map[string]bool

	sync.Mutex
}
