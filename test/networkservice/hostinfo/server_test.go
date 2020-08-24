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

package hostinfo_test

import (
	"context"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/mock"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/hostinfo"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/api/resourcepool"
)

func TestNewClient_AddHostInfo(t *testing.T) {
	defer goleak.VerifyNone(t)

	hostInfo := &resourcepool.HostInfo{
		HostName: "example.com",
		PhysicalFunctions: map[string]*resourcepool.PhysicalFunctionInfo{
			"0000:00:01:0": {
				Capability: "10G",
				IommuGroups: map[uint]*resourcepool.IommuGroupInfo{
					1: {
						DriverType:            resourcepool.NoDriver,
						TotalVirtualFunctions: 10,
						FreeVirtualFunctions:  5,
					},
				},
			},
		},
	}
	yamlHostInfo, _ := yaml.Marshal(hostInfo)

	hip := &hostInfoProviderMock{}
	hip.mock.On("GetHostInfo").
		Return(hostInfo)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}
	expected := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				ExtraContext: map[string]string{
					resourcepool.HostInfoKey: string(yamlHostInfo),
				},
			},
		},
	}
	client := next.NewNetworkServiceServer(hostinfo.NewServer(hip), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		assert.Equal(t, expected, request)
	}))

	_, err := client.Request(context.Background(), request)
	assert.Nil(t, err)
}

type hostInfoProviderMock struct {
	mock mock.Mock
}

func (hip *hostInfoProviderMock) GetHostInfo() *resourcepool.HostInfo {
	res := hip.mock.Called()
	return res.Get(0).(*resourcepool.HostInfo)
}
