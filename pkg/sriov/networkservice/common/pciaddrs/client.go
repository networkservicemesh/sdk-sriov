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

// Package pciaddrs contains chain element to fill available physical functions PCi addresses in host in the request
package pciaddrs

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

type setPCIAddressesClient struct {
	resourcePool *sriov.NetResourcePool
}

// NewClient - returns a new networkservicemesh.NetworkServiceClient that sets PCI addresses of all available on host
// SR-IOV capable net devices into the Request's SR-IOV and Kernel mechanisms
func NewClient(resourcePool *sriov.NetResourcePool) networkservice.NetworkServiceClient {
	return &setPCIAddressesClient{
		resourcePool: resourcePool,
	}
}

func (a *setPCIAddressesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	newMechanisms := make([]*networkservice.Mechanism, 0)

	for _, mech := range request.MechanismPreferences {
		// for each kernel or vfio mechanism add separate mechanisms with PCI addresses of available net devices
		if mech.GetType() == kernel.MECHANISM {
			mechsWithPci := a.withPCIAddresses(mech, kernel.PCIAddress)
			newMechanisms = append(newMechanisms, mechsWithPci...)
		} else if mech.GetType() == vfio.MECHANISM {
			mechsWithPci := a.withPCIAddresses(mech, vfio.PCIAddress)
			newMechanisms = append(newMechanisms, mechsWithPci...)
		}
	}

	request.MechanismPreferences = newMechanisms
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (a *setPCIAddressesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (a *setPCIAddressesClient) withPCIAddresses(mechanism *networkservice.Mechanism, pciAddressKey string) []*networkservice.Mechanism {
	var newMechanisms []*networkservice.Mechanism

	a.resourcePool.Lock()
	for _, res := range a.resourcePool.Resources {
		pciAddr := res.PhysicalFunction.PCIAddress

		newMech := mechanism.Clone()
		if newMech.GetParameters() == nil {
			newMech.Parameters = map[string]string{}
		}
		newMech.GetParameters()[pciAddressKey] = pciAddr
		newMechanisms = append(newMechanisms, newMech)
	}
	a.resourcePool.Unlock()

	return newMechanisms
}
