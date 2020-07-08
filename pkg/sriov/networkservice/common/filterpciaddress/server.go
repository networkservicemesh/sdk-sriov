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

// Package filterpciaddress provides a filtration of pci address
package filterpciaddress

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

///
// TODO refact code after PR is approved
const FreeVirtualFunctionsInfoKey string = "FreeVFs"

// TODO add HostName
type FreeVirtualFunctionsInfo struct {
	FreeVirtualFunctions map[string]int `yaml:"free_vfs"`
}

type filterPCIAddressServer struct {
	config     *sriov.Config
	freeVFInfo *FreeVirtualFunctionsInfo
}

func unmarshallFreeVirtualFunctionsInfo(config string) (*FreeVirtualFunctionsInfo, error) {
	stateConfig := &FreeVirtualFunctionsInfo{}

	rawBytes := []byte(config)
	if err := yaml.Unmarshal(rawBytes, stateConfig); err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling raw bytes %s", rawBytes)
	}

	return stateConfig, nil
}

///

// NewServer - filters out mechanisms by type and provided pci address parametr
func NewServer(config *sriov.Config) networkservice.NetworkServiceServer {
	return &filterPCIAddressServer{config: config, freeVFInfo: &FreeVirtualFunctionsInfo{FreeVirtualFunctions: make(map[string]int)}}
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

func (f *filterPCIAddressServer) isChecked(pciAddress string, cnt int) bool {
	if cnt > 0 {
		// get corresponding num of free virtual function on endpoint side
		epCnt, ok := f.freeVFInfo.FreeVirtualFunctions[pciAddress]
		if !ok { // first request
			f.freeVFInfo.FreeVirtualFunctions[pciAddress] = cnt
			return true
		} else if epCnt == cnt {
			return true
		}
		// TODO case1: epCnt > cnt
		// TODO case2: epCnt < cnt
	}

	return false
}

func (f *filterPCIAddressServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection() == nil {
		return nil, errors.New("request connection is invalid")
	}

	if request.GetConnection().GetContext() == nil {
		return nil, errors.New("connection context is invalid")
	}
	if request.GetConnection().GetContext().GetExtraContext() == nil {
		return nil, errors.New("connection extraContext is invalid")
	}

	// get pci address list with corresponding free virtual functions num
	strCfg := request.GetConnection().GetContext().GetExtraContext()[FreeVirtualFunctionsInfoKey]
	freeVirtualFunctionsInfo, err := unmarshallFreeVirtualFunctionsInfo(strCfg)
	if freeVirtualFunctionsInfo == nil {
		return nil, errors.Wrap(err, "FreeVirtualFunctionsInfo invalid")
	}

	// filtering
	var pciAddrList []string
	for pciAddr, cntFreeVF := range freeVirtualFunctionsInfo.FreeVirtualFunctions {
		if isSupportedPci(pciAddr, f.config.Domains) {
			if f.isChecked(pciAddr, cntFreeVF) {
				pciAddrList = append(pciAddrList, pciAddr)
			}
		}
	}

	if len(pciAddrList) == 0 {
		return nil, errors.New("received PCI address list is empty")
	}

	ctx = WithPCIAddrList(ctx, pciAddrList)
	ctx = WithFreeVFInfo(ctx, f.freeVFInfo)

	return next.Server(ctx).Request(ctx, request)
}

func (f *filterPCIAddressServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
