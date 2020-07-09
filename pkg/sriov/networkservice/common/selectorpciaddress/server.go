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

// Package selectorpciaddress provides a filtration of pci address
package selectorpciaddress

import (
	"context"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type selectPCIAddressServer struct {
	selector         *RoundRobinSelector
	config           *sriov.Config
	freeVFInfo       *FreeVirtualFunctionsInfo
	connectedPCIAddr map[string]string
	pciAddrKey       string
	sync.Mutex
}

// FreeVirtualFunctionsInfoKey key value for virtual functions info
const FreeVirtualFunctionsInfoKey string = "FreeVFs"

// FreeVirtualFunctionsInfo info about virtual functions
type FreeVirtualFunctionsInfo struct {
	// TODO add HostName
	FreeVirtualFunctions map[string]int `yaml:"free_vfs"`
}

func parseFreeVirtualFunctionsInfo(config string) (*FreeVirtualFunctionsInfo, error) {
	stateConfig := &FreeVirtualFunctionsInfo{}

	rawBytes := []byte(config)
	if err := yaml.Unmarshal(rawBytes, stateConfig); err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling raw bytes %s", rawBytes)
	}

	return stateConfig, nil
}

///

// NewServer - filters out mechanisms by type and provided pci address parametr
func NewServer(config *sriov.Config, pciAddrKey string) networkservice.NetworkServiceServer {
	return &selectPCIAddressServer{
		config:           config,
		connectedPCIAddr: map[string]string{},
		freeVFInfo:       &FreeVirtualFunctionsInfo{FreeVirtualFunctions: make(map[string]int)},
		selector:         NewRoundRobinSelector(),
		pciAddrKey:       pciAddrKey,
	}
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

// check on
func (s *selectPCIAddressServer) isChecked(pciAddress string, cntFreeVF int) bool {
	if cntFreeVF > 0 {
		// get corresponding num of free virtual function on endpoint side
		cntEP, ok := s.freeVFInfo.FreeVirtualFunctions[pciAddress]
		if !ok { // first request
			s.freeVFInfo.FreeVirtualFunctions[pciAddress] = cntFreeVF
			return true
		} else if cntEP == cntFreeVF {
			return true
		}
	}

	return false
}

// Get pci address
func getPCIAddrList(request *networkservice.NetworkServiceRequest) (*FreeVirtualFunctionsInfo, error) {
	strCfg := request.GetConnection().GetContext().GetExtraContext()[FreeVirtualFunctionsInfoKey]
	freeVirtualFunctionsInfo, err := parseFreeVirtualFunctionsInfo(strCfg)

	return freeVirtualFunctionsInfo, err
}

func isRequestValid(request *networkservice.NetworkServiceRequest) error {
	if request.GetConnection() == nil {
		return errors.New("request connection is invalid")
	}

	if request.GetConnection().GetContext() == nil {
		return errors.New("connection context is invalid")
	}
	if request.GetConnection().GetContext().GetExtraContext() == nil {
		return errors.New("connection extraContext is invalid")
	}

	return nil
}

// filtering pci address
func (s *selectPCIAddressServer) filterPCIAddr(freeVFInfo *FreeVirtualFunctionsInfo) ([]string, error) {
	var pciAddrList []string
	for pciAddr, cntFreeVF := range freeVFInfo.FreeVirtualFunctions {
		if isSupportedPci(pciAddr, s.config.Domains) {
			if s.isChecked(pciAddr, cntFreeVF) {
				pciAddrList = append(pciAddrList, pciAddr)
			}
		}
	}

	if len(pciAddrList) == 0 {
		return nil, errors.New("received PCI address list is not supported")
	}

	return pciAddrList, nil
}

func (s *selectPCIAddressServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.Lock()
	defer s.Unlock()

	if err := isRequestValid(request); err != nil {
		return nil, err
	}

	// get pci address list with corresponding free virtual functions num
	freeVirtualFunctionsInfo, err := getPCIAddrList(request)
	if err != nil {
		return nil, errors.Wrap(err, "FreeVirtualFunctionsInfo is invalid")
	}

	// filtering list
	pciAddrList, err := s.filterPCIAddr(freeVirtualFunctionsInfo)
	if err != nil {
		return nil, err
	}

	pciAddr := s.selector.SelectStringItem(pciAddrList)

	connection, err := next.Server(ctx).Request(ctx, request)

	if err == nil && pciAddr != "" {
		// set pci address for response
		if request.GetConnection().GetMechanism().GetParameters() == nil {
			request.GetConnection().GetMechanism().Parameters = map[string]string{}
		}
		request.GetConnection().GetMechanism().GetParameters()[s.pciAddrKey] = pciAddr
		// mark connection Id with corresponding pci address
		s.connectedPCIAddr[connection.GetId()] = pciAddr
		// decrement
		s.freeVFInfo.FreeVirtualFunctions[pciAddr]--
	}

	return connection, err
}

func (s *selectPCIAddressServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.Lock()
	defer s.Unlock()
	emptyValue, err := next.Server(ctx).Close(ctx, conn)
	if err == nil {
		if pciAddr, ok := s.connectedPCIAddr[conn.GetId()]; ok {
			// increment counter
			s.freeVFInfo.FreeVirtualFunctions[pciAddr]++

			// delete
			delete(s.connectedPCIAddr, conn.GetId())
		}
	}

	return emptyValue, err
}
