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

// Package filtermechanisms provides a filtration of preferences mechanisms
package filtermechanisms

import (
	"context"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type filterMechanismsServer struct {
	config   *sriov.Config
	mechUsed *sync.Map
}

// NewServer - filters out mechanisms by type and provided pci address parametr
func NewServer(config *sriov.Config, mechUsed *sync.Map) networkservice.NetworkServiceServer {
	return &filterMechanismsServer{config: config, mechUsed: mechUsed}
}

// Search pci device in config
// TODO add hostName, capability as parameters for filtering
func isSupportedPci(pciAddress string, domains []sriov.ResourceDomain) bool {
	for _, domain := range domains {
		for _, pciDevice := range domain.PCIDevices {
			if pciDevice.PCIAddress == pciAddress {
				return true
			}
		}
	}

	return false
}

func (f *filterMechanismsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection() == nil {
		request.Connection = &networkservice.Connection{Id: "Id"}
	}
	// check on contains
	if mech, ok := f.mechUsed.Load(request.GetConnection().GetId()); ok {
		request.Connection.Mechanism = mech.(*networkservice.Mechanism)
		return request.GetConnection(), nil
	}

	var mechanisms []*networkservice.Mechanism
	for _, mechanism := range request.GetMechanismPreferences() {
		pciAddress := ""
		switch mechanism.GetType() {
		case kernel.MECHANISM:
			pciAddress = kernel.ToMechanism(mechanism).GetPCIAddress()
		case vfio.MECHANISM:
			pciAddress = vfio.ToMechanism(mechanism).GetPCIAddress()
		}

		if pciAddress != "" && isSupportedPci(pciAddress, f.config.Domains) {
			mechanisms = append(mechanisms, mechanism)
		}
	}
	request.MechanismPreferences = mechanisms

	return next.Server(ctx).Request(ctx, request)
}

func (f *filterMechanismsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
