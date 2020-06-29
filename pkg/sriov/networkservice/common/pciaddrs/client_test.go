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

package pciaddrs_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/common/pciaddrs"
)

func TestNewClient_AddKernelMechanisms(t *testing.T) {
	defer goleak.VerifyNone(t)

	resourcePool := &sriov.NetResourcePool{
		Resources: []*sriov.NetResource{
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: "0000:00:01:0",
				},
			},
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: "0000:00:02:0",
				},
			},
		},
	}
	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: kernel.MECHANISM,
			},
		},
	}
	expected := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					kernel.PCIAddress: "0000:00:01:0",
				},
			},
			{
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					kernel.PCIAddress: "0000:00:02:0",
				},
			},
		},
	}
	client := next.NewNetworkServiceClient(pciaddrs.NewClient(resourcePool), checkrequest.NewClient(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		assert.Equal(t, expected, request)
	}))

	_, err := client.Request(context.Background(), request)
	assert.Nil(t, err)
}

func TestNewClient_AddVFIOMechanisms(t *testing.T) {
	defer goleak.VerifyNone(t)

	resourcePool := &sriov.NetResourcePool{
		Resources: []*sriov.NetResource{
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: "0000:00:01:0",
				},
			},
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: "0000:00:02:0",
				},
			},
		},
	}
	expected := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: vfio.MECHANISM,
				Parameters: map[string]string{
					kernel.PCIAddress: "0000:00:01:0",
				},
			},
			{
				Type: vfio.MECHANISM,
				Parameters: map[string]string{
					kernel.PCIAddress: "0000:00:02:0",
				},
			},
		},
	}
	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: vfio.MECHANISM,
			},
		},
	}
	client := next.NewNetworkServiceClient(pciaddrs.NewClient(resourcePool), checkrequest.NewClient(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		assert.Equal(t, expected, request)
	}))

	_, err := client.Request(context.Background(), request)
	assert.Nil(t, err)
}
