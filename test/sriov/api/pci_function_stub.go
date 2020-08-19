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

// Package api_test provides stubs for the api/...
package api_test

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/api/pcifunction"
)

// PCIFunctionFactoryStub is a stub for pcifunction.Factory
type PCIFunctionFactoryStub struct {
	Pfs []*PCIPhysicalFunctionStub `yaml:"pfs"`
}

// NewPhysicalFunction returns a PCIPhysicalFunctionStub with the given pciAddress
func (pciff *PCIFunctionFactoryStub) NewPhysicalFunction(pciAddress string) (pcifunction.PhysicalFunction, error) {
	for _, pf := range pciff.Pfs {
		if pf.Addr == pciAddress {
			return pf, nil
		}
	}
	return nil, errors.Errorf("device doesn't exist: %v", pciAddress)
}

// PCIPhysicalFunctionStub is a stub for pcifunction.PhysicalFunction
type PCIPhysicalFunctionStub struct {
	Capacity int                `yaml:"capacity"`
	Vfs      []*PCIFunctionStub `yaml:"vfs"`

	PCIFunctionStub
}

// GetVirtualFunctions returns pcipf.Vfs
func (pcipf *PCIPhysicalFunctionStub) GetVirtualFunctions() ([]pcifunction.BindablePCIFunction, error) {
	var vfs []pcifunction.BindablePCIFunction
	for _, vf := range pcipf.Vfs {
		vfs = append(vfs, vf)
	}
	return vfs, nil
}

// PCIFunctionStub is a stub for pcifunction.BindableFunction
type PCIFunctionStub struct {
	Addr       string `yaml:"addr"`
	IfName     string `yaml:"ifName"`
	IommuGroup uint   `yaml:"iommuGroup"`
	Driver     string `yaml:"driver"`
}

// GetPCIAddress returns pcif.Addr
func (pcif *PCIFunctionStub) GetPCIAddress() string {
	return pcif.Addr
}

// GetNetInterfaceName returns pcif.IfName
func (pcif *PCIFunctionStub) GetNetInterfaceName() (string, error) {
	return pcif.IfName, nil
}

// GetIommuGroupID returns pcif.IommuGroup
func (pcif *PCIFunctionStub) GetIommuGroupID() (uint, error) {
	return pcif.IommuGroup, nil
}

// GetBoundDriver returns pcif.Driver
func (pcif *PCIFunctionStub) GetBoundDriver() (string, error) {
	return pcif.Driver, nil
}

// BindDriver sets pcif.Driver = driver
func (pcif *PCIFunctionStub) BindDriver(driver string) error {
	pcif.Driver = driver
	return nil
}
