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
	"fmt"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"runtime"
	"testing"

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
				kernelMech.PCIAddress: pfPCIAddress,
			},
		},
	}
	expected := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernelMech.MECHANISM,
			Parameters: map[string]string{
				kernelMech.PCIAddress:       pfPCIAddress,
				kernelMech.InterfaceNameKey: vf2IfaceName,
			},
		},
	}

	server := next.NewNetworkServiceServer(kernel.NewServer(resourcePool), mockedEndpoint{conn: fromEndpoint})
	conn, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
	assert.Equal(t, expected, conn)

	selectedVfState := resourcePool.Resources[0].PhysicalFunction.VirtualFunctions[vf2]
	assert.Equal(t, sriov.UsedVirtualFunction, selectedVfState)
}

func TestName(t *testing.T) {
	// Lock the OS Thread so we don't accidentally switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save the current network namespace
	origns, _ := netns.Get()
	defer origns.Close()
	ifaces, _ := net.Interfaces()
	fmt.Printf("Interfaces: %v\n", ifaces)

	// Create a new network namespace
	newns, _ := netns.New()
	defer newns.Close()
	ifaces, _ = net.Interfaces()
	fmt.Printf("Interfaces: %v\n", ifaces)

	// Switch back to the original namespace
	netns.Set(origns)

	netlink.NewHandle()

	//newNsFdInt := int(newns)
	//println(newNsFdInt)
	//
	//newNsFd := uintptr(newns)
	//println(newNsFd)

}

//func TestNewServer_NoFreeVirtualFunctions(t *testing.T) {
//	defer goleak.VerifyNone(t)
//
//	vf1, vf2 := initVfs()
//	resourcePool := &sriov.NetResourcePool{
//		Resources: []*sriov.NetResource{
//			{
//				PhysicalFunction: &sriov.PhysicalFunction{
//					PCIAddress: pfPCIAddress,
//					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
//						vf1: sriov.UsedVirtualFunction,
//						vf2: sriov.UsedVirtualFunction,
//					},
//				},
//			},
//		},
//	}
//	fromEndpoint := &networkservice.Connection{
//		Mechanism: &networkservice.Mechanism{
//			Type: kernelMech.MECHANISM,
//			Parameters: map[string]string{
//				kernelMech.PCIAddress: pfPCIAddress,
//			},
//		},
//	}
//
//	server := next.NewNetworkServiceServer(kernel.NewServer(resourcePool), mockedEndpoint{conn: fromEndpoint})
//	conn, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{})
//	assert.Nil(t, conn)
//	assert.NotNil(t, err)
//}
//
//func TestNewServer_ReleaseVirtualFunctions(t *testing.T) {
//	defer goleak.VerifyNone(t)
//
//	vf1, vf2 := initVfs()
//	resourcePool := &sriov.NetResourcePool{
//		Resources: []*sriov.NetResource{
//			{
//				PhysicalFunction: &sriov.PhysicalFunction{
//					PCIAddress: pfPCIAddress,
//					VirtualFunctions: map[*sriov.VirtualFunction]sriov.VirtualFunctionState{
//						vf1: sriov.UsedVirtualFunction,
//						vf2: sriov.UsedVirtualFunction,
//					},
//				},
//			},
//		},
//	}
//	conn := &networkservice.Connection{
//		Mechanism: &networkservice.Mechanism{
//			Type: kernelMech.MECHANISM,
//			Parameters: map[string]string{
//				kernelMech.PCIAddress:       pfPCIAddress,
//				kernelMech.InterfaceNameKey: vf1IfaceName,
//			},
//		},
//	}
//
//	client := next.NewNetworkServiceServer(kernel.NewServer(resourcePool))
//	_, err := client.Close(context.Background(), conn)
//	assert.Nil(t, err)
//
//	freedVfState := resourcePool.Resources[0].PhysicalFunction.VirtualFunctions[vf1]
//	assert.Equal(t, sriov.FreeVirtualFunction, freedVfState)
//}
