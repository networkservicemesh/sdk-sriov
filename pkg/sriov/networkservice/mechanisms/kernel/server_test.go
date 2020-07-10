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

package kernel_test

import (
	"context"
	"testing"

	"github.com/vishvananda/netns"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/mechanisms/kernel"
)

const (
	pfPCIAddress  = "0000:01:00:0"
	vf1PCIAddress = "0000:01:00:1"
	vf1IfaceName  = "enp1s1"
	vf2PCIAddress = "0000:01:00:2"
	vf2IfaceName  = "enp1s2"
)

type mockedEndpoint struct {
	conn *networkservice.Connection
}

func (m mockedEndpoint) Request(context.Context, *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return m.conn, nil
}

func (m mockedEndpoint) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

type mockedKernelProvider struct{}

func (m mockedKernelProvider) MoveInterfaceToAnotherNamespace(string, netns.NsHandle, netns.NsHandle) error {
	return nil
}

func (m mockedKernelProvider) GetNSHandleFromInode(string) (netns.NsHandle, error) {
	return 0, nil
}

func initVfs() (vf1, vf2 *sriov.VirtualFunction) {
	vf1 = &sriov.VirtualFunction{
		PCIAddress:       vf1PCIAddress,
		NetInterfaceName: vf1IfaceName,
	}
	vf2 = &sriov.VirtualFunction{
		PCIAddress:       vf2PCIAddress,
		NetInterfaceName: vf2IfaceName,
	}
	return
}

func TestNewServer_SelectVirtualFunction(t *testing.T) {
	defer goleak.VerifyNone(t)

	vf1, vf2 := initVfs()
	resourcePool := &sriov.NetResourcePool{
		Resources: []*sriov.NetResource{
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: pfPCIAddress,
					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
						vf1: sriov.UsedVirtualFunction,
						vf2: sriov.FreeVirtualFunction,
					},
				},
			},
		},
	}
	fromEndpoint := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernelMech.MECHANISM,
			Parameters: map[string]string{
				kernelMech.PCIAddress:    pfPCIAddress,
				kernelMech.NetNSInodeKey: "0",
			},
		},
	}
	expected := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernelMech.MECHANISM,
			Parameters: map[string]string{
				kernelMech.PCIAddress:       pfPCIAddress,
				kernelMech.NetNSInodeKey:    "0",
				kernelMech.InterfaceNameKey: vf2IfaceName,
			},
		},
	}

	server := next.NewNetworkServiceServer(kernel.NewServer(resourcePool, mockedKernelProvider{}), mockedEndpoint{conn: fromEndpoint})
	conn, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
	assert.Equal(t, expected, conn)

	selectedVfState := resourcePool.Resources[0].PhysicalFunction.VirtualFunctions[vf2]
	assert.Equal(t, sriov.UsedVirtualFunction, selectedVfState)
}

func TestNewServer_NoFreeVirtualFunctions(t *testing.T) {
	defer goleak.VerifyNone(t)

	vf1, vf2 := initVfs()
	resourcePool := &sriov.NetResourcePool{
		Resources: []*sriov.NetResource{
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: pfPCIAddress,
					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
						vf1: sriov.UsedVirtualFunction,
						vf2: sriov.UsedVirtualFunction,
					},
				},
			},
		},
	}
	fromEndpoint := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernelMech.MECHANISM,
			Parameters: map[string]string{
				kernelMech.PCIAddress:    pfPCIAddress,
				kernelMech.NetNSInodeKey: "0",
			},
		},
	}

	server := next.NewNetworkServiceServer(kernel.NewServer(resourcePool, mockedKernelProvider{}), mockedEndpoint{conn: fromEndpoint})
	conn, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestNewServer_ReleaseVirtualFunctions(t *testing.T) {
	defer goleak.VerifyNone(t)

	vf1, vf2 := initVfs()
	resourcePool := &sriov.NetResourcePool{
		Resources: []*sriov.NetResource{
			{
				PhysicalFunction: &sriov.PhysicalFunction{
					PCIAddress: pfPCIAddress,
					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
						vf1: sriov.UsedVirtualFunction,
						vf2: sriov.UsedVirtualFunction,
					},
				},
			},
		},
	}
	conn := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernelMech.MECHANISM,
			Parameters: map[string]string{
				kernelMech.PCIAddress:       pfPCIAddress,
				kernelMech.NetNSInodeKey:    "0",
				kernelMech.InterfaceNameKey: vf1IfaceName,
			},
		},
	}

	client := next.NewNetworkServiceServer(kernel.NewServer(resourcePool, mockedKernelProvider{}))
	_, err := client.Close(context.Background(), conn)
	assert.Nil(t, err)

	freedVfState := resourcePool.Resources[0].PhysicalFunction.VirtualFunctions[vf1]
	assert.Equal(t, sriov.FreeVirtualFunction, freedVfState)
}
