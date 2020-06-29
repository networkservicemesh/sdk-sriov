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
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"google.golang.org/grpc"
)

type safeIndex struct {
	index int
	sync.Mutex
}

type safePciInfo struct {
	info map[string]*networkservice.Mechanism
	sync.Mutex
}

type nseImpl struct {
	server    *grpc.Server
	ctx       context.Context
	cancel    context.CancelFunc
	listenOn  *url.URL
	config    *sriov.Config
	pciIndex  safeIndex
	pciUsed   safePciInfo
	errorChan <-chan error
}

func (s *safePciInfo) Add(connID string, mech *networkservice.Mechanism) {
	s.Lock()
	s.info[connID] = mech
	s.Unlock()
}

func (s *safePciInfo) Remove(connID string) {
	s.Lock()
	delete(s.info, connID)
	s.Unlock()
}

// NewServer a new endpoint and running on grpc server
func NewServer(ctx context.Context, listenOn *url.URL, config *sriov.Config) (server *grpc.Server, errChan <-chan error) {
	// if we havn't config file then endpoint will not start
	if config == nil {
		errChan := make(chan error, 1)
		errChan <- errors.New("empty config")
		return nil, errChan
	}

	nse := &nseImpl{
		listenOn: listenOn,
		config:   config,
		pciIndex: safeIndex{index: 0},
		pciUsed:  safePciInfo{info: map[string]*networkservice.Mechanism{}},
		server:   grpc.NewServer(),
	}

	networkservice.RegisterNetworkServiceServer(nse.server, nse)

	nse.ctx, nse.cancel = context.WithCancel(ctx)
	nse.errorChan = grpcutils.ListenAndServe(nse.ctx, nse.listenOn, nse.server)

	return nse.server, nse.errorChan
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

// getFilteredMechanisms return list of pci address filtered by supported mechanism
func getFilteredMechanisms(mechList []*networkservice.Mechanism, domains []sriov.ResourceDomain) (mechListFiltered []*networkservice.Mechanism) {
	var pciAddress string
	for _, mech := range mechList {
		pciAddress = ""
		switch mech.GetType() {
		case kernel.MECHANISM:
			pciAddress = kernel.ToMechanism(mech).GetPCIAddress()
		case vfio.MECHANISM:
			pciAddress = vfio.ToMechanism(mech).GetPCIAddress()
		}

		if pciAddress != "" && isSupportedPci(pciAddress, domains) {
			mechListFiltered = append(mechListFiltered, mech)
		}
	}

	return
}

func selectMech(s *safeIndex, mechList []*networkservice.Mechanism) (mech *networkservice.Mechanism) {
	s.Lock()
	newIndex := s.index % len(mechList)
	mech = mechList[newIndex]
	s.index = newIndex + 1
	s.Unlock()

	return
}

func (d *nseImpl) Request(_ context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Mechanism.Parameters = map[string]string{}

	// get pci address list for selection
	mechList := getFilteredMechanisms(request.GetMechanismPreferences(), d.config.Domains)
	if mechList != nil {
		mech := selectMech(&d.pciIndex, mechList)
		request.Connection.Mechanism = mech
		// TODO allocate resources
		d.pciUsed.Add(request.GetConnection().Id, mech)
		return request.GetConnection(), nil
	}

	return request.GetConnection(), errors.New("specified ports are not supported")
}

func (d *nseImpl) Close(_ context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// TODO release resources
	d.pciUsed.Remove(conn.Id)
	return &empty.Empty{}, nil
}
