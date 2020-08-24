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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vfioapi "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/test/stub"
)

const (
	clientCgroupDirKey = "clientCgroupDir"
	deviceAllowFile    = "devices.allow"
	deviceDenyFile     = "devices.deny"
	deviceStringFormat = "c %d:%d rwm"
	vfioMajorKey       = "vfioMajor"
	vfioMinorKey       = "vfioMinor"
	deviceMajorKey     = "deviceMajor"
	deviceMinorKey     = "deviceMinor"
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
	)

	err = stub.InputFileAPI(ctx, path.Join(tmpDir, deviceAllowFile), func(s string) {
		var major, minor int
		_, _ = fmt.Sscanf(s, deviceStringFormat, &major, &minor)
		allowedDevices.Lock()
		allowedDevices.devices[fmt.Sprintf("%d:%d", major, minor)] = true
		allowedDevices.Unlock()
	})
	assert.Nil(t, err)
	err = stub.InputFileAPI(ctx, path.Join(tmpDir, deviceDenyFile), func(s string) {
		var major, minor int
		_, _ = fmt.Sscanf(s, deviceStringFormat, &major, &minor)
		allowedDevices.Lock()
		delete(allowedDevices.devices, fmt.Sprintf("%d:%d", major, minor))
		allowedDevices.Unlock()
	})
	assert.Nil(t, err)

	return server, tmpDir
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
		vfioMajorKey:   "1",
		vfioMinorKey:   "2",
		deviceMajorKey: "3",
		deviceMinorKey: "4",
	}

	_, err := server.Close(ctx, conn)
	assert.Nil(t, err)

	assert.Eventually(t, func() bool {
		allowedDevices.Lock()
		defer allowedDevices.Unlock()
		return reflect.DeepEqual(map[string]bool{}, allowedDevices.devices)
	}, time.Second, 10*time.Millisecond)
}

type allowedDevices struct {
	devices map[string]bool

	sync.Mutex
}
