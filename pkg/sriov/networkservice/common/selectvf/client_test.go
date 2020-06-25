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

package selectvf_test

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/common/selectvf"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/types"
	"github.com/networkservicemesh/sdk-sriov/test/mocks"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/goleak"
	"testing"
)

type mockedEndpoint struct {
	conn *networkservice.Connection
}

func (m mockedEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return m.conn, nil
}

func (m mockedEndpoint) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

type vfData struct {
	pciAddr      string
	netIfaceName string
}

type pfData struct {
	pciAddr        string
	setVfInUseChan chan types.VirtualFunction
	setVfFreeChan  chan types.VirtualFunction
	freeVfs        []*vfData
	vfsInUse       []*vfData
}

func getMockedResourcePool(pfs []*pfData) types.NetResourcePool {
	netResources := make([]types.NetResource, 0)
	for _, pf := range pfs {
		physicalFunction := new(mocks.PhysicalFunction)
		physicalFunction.On("GetPCIAddress").Return(pf.pciAddr)

		physicalFunction.On("SetVirtualFunctionFree", mock.Anything).Run(func(args mock.Arguments) {
			pf.setVfFreeChan <- args.Get(0).(types.VirtualFunction)
		}).Return(pf.pciAddr)

		physicalFunction.On("SetVirtualFunctionInUse", mock.Anything).Run(func(args mock.Arguments) {
			pf.setVfInUseChan <- args.Get(0).(types.VirtualFunction)
		}).Return(pf.pciAddr)

		freeVfs := make([]types.VirtualFunction, 0)
		for _, vf := range pf.freeVfs {
			virtualFunction := new(mocks.VirtualFunction)
			virtualFunction.On("GetPCIAddress").Return(vf.pciAddr)
			virtualFunction.On("GetNetInterfaceName").Return(vf.netIfaceName)

			freeVfs = append(freeVfs, virtualFunction)
		}
		physicalFunction.On("GetFreeVirtualFunctions").Return(freeVfs)

		vfsInUse := make([]types.VirtualFunction, 0)
		for _, vf := range pf.vfsInUse {
			virtualFunction := new(mocks.VirtualFunction)
			virtualFunction.On("GetPCIAddress").Return(vf.pciAddr)
			virtualFunction.On("GetNetInterfaceName").Return(vf.netIfaceName)

			vfsInUse = append(vfsInUse, virtualFunction)
		}
		physicalFunction.On("GetVirtualFunctionsInUse").Return(vfsInUse)

		netResource := new(mocks.NetResource)
		netResource.On("GetPhysicalFunction").Return(physicalFunction)

		netResources = append(netResources, netResource)
	}

	netResourcePool := new(mocks.NetResourcePool)
	netResourcePool.On("GetResources").Return(netResources)

	return netResourcePool
}

func TestNewClient_SelectVirtualFunction(t *testing.T) {
	defer goleak.VerifyNone(t)

	pfPciAddress := "0000:01:00:0"
	setVfInUseChan := make(chan types.VirtualFunction, 1)
	selectedVfPciAddr := "0000:01:00:1"
	selectedVfNetIfaceName := "enp1s1"
	resourcePool := sriov.NetResourcePool{}
	pfs := []*pfData{
		{
			pciAddr:        pfPciAddress,
			setVfInUseChan: setVfInUseChan,
			freeVfs: []*vfData{
				{
					pciAddr:      selectedVfPciAddr,
					netIfaceName: selectedVfNetIfaceName,
				},
				{
					pciAddr:      "0000:01:00:2",
					netIfaceName: "enp1s2",
				},
			},
		},
	}
	resourcePool := getMockedResourcePool(pfs)
	fromEndpoint := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernel.MECHANISM,
			Parameters: map[string]string{
				kernel.PCIAddress: pfPciAddress,
			},
		},
	}
	expected := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernel.MECHANISM,
			Parameters: map[string]string{
				kernel.PCIAddress:       pfPciAddress,
				kernel.InterfaceNameKey: selectedVfNetIfaceName,
			},
		},
	}

	client := next.NewNetworkServiceClient(selectvf.NewClient(resourcePool), adapters.NewServerToClient(mockedEndpoint{conn: fromEndpoint}))
	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Equal(t, expected, conn)
	assert.Nil(t, err)

	setInUseVf := <-setVfInUseChan
	assert.Equal(t, selectedVfPciAddr, setInUseVf.GetPCIAddress())
	assert.Equal(t, selectedVfNetIfaceName, setInUseVf.GetNetInterfaceName())
}

func TestNewClient_NoFreeVirtualFunctions(t *testing.T) {
	defer goleak.VerifyNone(t)

	pfPciAddress := "0000:01:00:0"
	pfs := []*pfData{
		{
			pciAddr: pfPciAddress,
			freeVfs: []*vfData{}, // no free virtual functions
		},
	}
	resourcePool := getMockedResourcePool(pfs)
	fromEndpoint := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernel.MECHANISM,
			Parameters: map[string]string{
				kernel.PCIAddress: pfPciAddress,
			},
		},
	}

	client := next.NewNetworkServiceClient(selectvf.NewClient(resourcePool), adapters.NewServerToClient(mockedEndpoint{conn: fromEndpoint}))
	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestNewClient_FreeVirtualFunctionsOnClose(t *testing.T) {
	defer goleak.VerifyNone(t)

	pfPciAddress := "0000:01:00:0"
	setVfFreeChan := make(chan types.VirtualFunction, 1)
	releasedVfPciAddr := "0000:01:00:2"
	releasedVfNetIfaceName := "enp1s2"
	pfs := []*pfData{
		{
			pciAddr:       pfPciAddress,
			setVfFreeChan: setVfFreeChan,
			vfsInUse: []*vfData{
				{
					pciAddr:      "0000:01:00:1",
					netIfaceName: "enp1s1",
				},
				{
					pciAddr:      releasedVfPciAddr,
					netIfaceName: releasedVfNetIfaceName,
				},
			},
		},
	}
	resourcePool := getMockedResourcePool(pfs)
	conn := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernel.MECHANISM,
			Parameters: map[string]string{
				kernel.PCIAddress:       pfPciAddress,
				kernel.InterfaceNameKey: releasedVfNetIfaceName,
			},
		},
	}

	client := next.NewNetworkServiceClient(selectvf.NewClient(resourcePool))
	_, err := client.Close(context.Background(), conn)
	assert.Nil(t, err)

	setFreeVf := <-setVfFreeChan
	assert.Equal(t, releasedVfPciAddr, setFreeVf.GetPCIAddress())
	assert.Equal(t, releasedVfNetIfaceName, setFreeVf.GetNetInterfaceName())
}
