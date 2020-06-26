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

// Package selectvf provides chain element for selection one of the available virtual functions for client based on
// returned PCI address from endpoint
package selectvf

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

type selectVirtualFunctionClient struct {
	resourcePool *sriov.NetResourcePool
}

// NewClient - returns a new authorization networkservicemesh.NetworkServiceClient
func NewClient(resourcePool *sriov.NetResourcePool) networkservice.NetworkServiceClient {
	return &selectVirtualFunctionClient{
		resourcePool: resourcePool,
	}
}

func (a *selectVirtualFunctionClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	var pciAddress string
	var ok bool
	switch conn.GetMechanism().GetType() {
	case kernel.MECHANISM:
		pciAddress, ok = conn.GetMechanism().GetParameters()[kernel.PCIAddress]
		if !ok {
			return nil, errors.Errorf("No selected PCI address provided")
		}
	case vfio.MECHANISM:
		pciAddress, ok = conn.GetMechanism().GetParameters()[vfio.PCIAddress]
		if !ok {
			return nil, errors.Errorf("No selected PCI address provided")
		}
	default:
		return conn, nil
	}

	selectedVf, err := a.selectVirtualFunction(pciAddress)
	if err != nil {
		return nil, err
	}

	if conn.GetMechanism().GetType() == kernel.MECHANISM {
		conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey] = selectedVf.NetInterfaceName
	}
	// TODO - add VFIO-specific info about selected virtual function - e.g. VirtualFunction PCIAddress

	return conn, nil
}

func (a *selectVirtualFunctionClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if conn.GetMechanism().GetType() == kernel.MECHANISM {
		pciAddress, ok := conn.GetMechanism().GetParameters()[kernel.PCIAddress]
		if !ok {
			return nil, errors.Errorf("No physical function PCI address found")
		}
		netIfaceName, ok := conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey]
		if !ok {
			return nil, errors.Errorf("No net interface name found")
		}
		err := a.freeVirtualFunction(pciAddress, netIfaceName)
		if err != nil {
			return nil, err
		}
	}
	// TODO - get VFIO-specific info about selected virtual function - e.g. VirtualFunction PCIAddress

	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (a *selectVirtualFunctionClient) selectVirtualFunction(pfPCIAddr string) (selectedVf *sriov.VirtualFunction, err error) {
	a.resourcePool.Lock()
	defer a.resourcePool.Unlock()

	for _, netResource := range a.resourcePool.Resources {
		pf := netResource.PhysicalFunction
		if pf.PCIAddress != pfPCIAddr {
			continue
		}

		// select the first free virtual function
		for vf, state := range pf.VirtualFunctions {
			if state == sriov.FreeVirtualFunction {
				selectedVf = vf
				break
			}
		}
		if selectedVf == nil {
			return nil, errors.Errorf("no free virtual function found for device %s", pfPCIAddr)
		}

		// mark it as in use
		err = pf.SetVirtualFunctionState(selectedVf, sriov.UsedVirtualFunction)
		if err != nil {
			return nil, err
		}
		return selectedVf, nil
	}
	return nil, errors.Errorf("no physical function with PCI address %s found", pfPCIAddr)
}

func (a *selectVirtualFunctionClient) freeVirtualFunction(pfPCIAddr, vfNetIfaceName string) error {
	a.resourcePool.Lock()
	defer a.resourcePool.Unlock()

	for _, netResource := range a.resourcePool.Resources {
		pf := netResource.PhysicalFunction
		if pf.PCIAddress != pfPCIAddr {
			continue
		}

		for vf := range pf.VirtualFunctions {
			if vf.NetInterfaceName == vfNetIfaceName {
				return pf.SetVirtualFunctionState(vf, sriov.FreeVirtualFunction)
			}
		}
		return errors.Errorf("no virtual function with net interface name %s found", vfNetIfaceName)
	}
	return errors.Errorf("no physical function with PCI address %s found", pfPCIAddr)
}
