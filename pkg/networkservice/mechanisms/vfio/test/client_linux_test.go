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
	"os"
	"path"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/mechanisms/vfio"
)

func TestVfioClient_Request(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	tmpDir := path.Join(os.TempDir(), t.Name())
	err := os.MkdirAll(tmpDir, 0750)
	assert.Nil(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	client := chain.NewNetworkServiceClient(
		vfio.NewClient(tmpDir, cgroupDir),
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					vfio.IommuGroupKey: iommuGroup,
					vfioMajorKey:       "1",
					vfioMinorKey:       "2",
					deviceMajorKey:     "3",
					deviceMinorKey:     "4",
				},
			},
			Context: &networkservice.ConnectionContext{
				ExtraContext: map[string]string{},
			},
		},
	})
	assert.Nil(t, err)

	assert.Equal(t, cgroupDir, conn.Context.ExtraContext[clientCgroupDirKey])

	info := new(unix.Stat_t)

	err = unix.Stat(path.Join(tmpDir, vfioDevice), info)
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), unix.Major(info.Rdev))
	assert.Equal(t, uint32(2), unix.Minor(info.Rdev))

	err = unix.Stat(path.Join(tmpDir, iommuGroup), info)
	assert.Nil(t, err)
	assert.Equal(t, uint32(3), unix.Major(info.Rdev))
	assert.Equal(t, uint32(4), unix.Minor(info.Rdev))
}
