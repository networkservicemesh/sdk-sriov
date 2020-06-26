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

// Package endpoint define a test endpoint listening on passed URL.
package endpoint

import (
	"context"
	"errors"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"google.golang.org/grpc"
)

const (
	configFileName = "config.yml"
)

type nseImpl struct {
	server    *grpc.Server
	ctx       context.Context
	cancel    context.CancelFunc
	listenOn  *url.URL
	errorChan <-chan error
}

type pciInfo struct {
	isKernel   bool // flag for selection of corresponding mechanism
	pciAddress string
}

var pciIndex int
var config *sriov.Config

// NewServer a new endpoint and running on grpc server
func NewServer(ctx context.Context, listenOn *url.URL) (server *grpc.Server, errChan <-chan error) {
	// if we havn't config file then endpoint will not start
	var err error
	config, err = sriov.ReadConfig(ctx, configFileName)
	if err != nil {
		errCh2 := make(chan error, 1)
		errCh2 <- err
		return nil, errCh2
	}

	pciIndex = 0
	nse := &nseImpl{
		listenOn: listenOn,
		server:   grpc.NewServer(),
	}

	networkservice.RegisterNetworkServiceServer(nse.server, nse)

	nse.ctx, nse.cancel = context.WithCancel(ctx)
	nse.errorChan = grpcutils.ListenAndServe(nse.ctx, nse.listenOn, nse.server)

	return nse.server, nse.errorChan
}

// Search pci device in config
// TODO add hostName as parameter for filtering
func isSupportedPci(pciAddress string) bool {
	for _, domain := range config.Domains {
		for _, pciDevice := range domain.PCIDevices {
			if pciDevice.PCIAddress == pciAddress {
				return true
			}
		}
	}

	return false
}

// getPciListByMechanisms return list of pci address filtered by supported mechanism
func getPciListByMechanisms(mechanisms []*networkservice.Mechanism) (list []pciInfo) {
	var pciAddress string
	var isKernel bool
	for _, mech := range mechanisms {
		pciAddress = ""
		isKernel = true
		switch mech.GetType() {
		case kernel.MECHANISM:
			pciAddress = kernel.ToMechanism(mech).GetPCIAddress()
		case vfio.MECHANISM:
			pciAddress = vfio.ToMechanism(mech).GetPCIAddress()
			isKernel = false
		}

		if pciAddress != "" {
			if isSupportedPci(pciAddress) {
				list = append(list, pciInfo{isKernel: isKernel, pciAddress: pciAddress})
			}
		}
	}

	return list
}

func selectPciInfo(list []pciInfo) (info pciInfo) {
	index := pciIndex % len(list)
	info = list[index]
	pciIndex++

	return info
}

func (d *nseImpl) Request(_ context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Mechanism.Parameters = map[string]string{}

	// get pci address list for selection
	list := getPciListByMechanisms(request.GetMechanismPreferences())
	if list != nil {
		info := selectPciInfo(list)
		key := kernel.PCIAddress
		if !info.isKernel {
			key = vfio.PCIAddress
		}
		request.Connection.Mechanism.Parameters = map[string]string{key: info.pciAddress}

		return request.GetConnection(), nil
	}

	return request.GetConnection(), errors.New("specified ports are not supported")
}

func (d *nseImpl) Close(_ context.Context, _ *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
