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
	"testing"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/mechanisms/vfio"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vfioMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

func TestNewClient_AddStateConfig(t *testing.T) {
	defer goleak.VerifyNone(t)

	resourcePool := &sriov.NetResourcePool{
		Resources: []*sriov.NetResource{
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: "0000:00:01:0",
					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
						{}: sriov.FreeVirtualFunction,
						{}: sriov.FreeVirtualFunction,
					},
				},
			},
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: "0000:00:02:0",
					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
						{}: sriov.FreeVirtualFunction,
						{}: sriov.UsedVirtualFunction,
					},
				},
			},
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: "0000:00:03:0",
					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
						{}: sriov.UsedVirtualFunction,
						{}: sriov.UsedVirtualFunction,
					},
				},
			},
		},
	}
	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: vfioMech.MECHANISM,
			},
		},
	}
	expected := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: vfioMech.MECHANISM,
			},
		},
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				ExtraContext: map[string]string{
					sriov.VirtualFunctionsStateConfigKey: "Config:\n  \"0000:00:01:0\": 2\n  \"0000:00:02:0\": 1\n  \"0000:00:03:0\": 0\n",
				},
			},
		},
	}
	client := next.NewNetworkServiceServer(vfio.NewServer(resourcePool), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		assert.Equal(t, expected, request)
	}))

	_, err := client.Request(context.Background(), request)
	assert.Nil(t, err)
}
