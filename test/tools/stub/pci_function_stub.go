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

// Package stub provides stubs for testing
package stub

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/tools/api/pcifunction"
)

// PCIFunctionFactory is a stub for pcifunction.Factory
type PCIFunctionFactory struct {
	Pfs []*PCIPhysicalFunction `yaml:"pfs"`
}

// NewPhysicalFunction returns a PCIPhysicalFunction with the given pciAddress
func (pciff *PCIFunctionFactory) NewPhysicalFunction(pciAddress string) (pcifunction.PhysicalFunction, error) {
	for _, pf := range pciff.Pfs {
		if pf.Addr == pciAddress {
			return pf, nil
		}
	}
	return nil, errors.Errorf("device doesn't exist: %v", pciAddress)
}

// PCIPhysicalFunction is a stub for pcifunction.PhysicalFunction
type PCIPhysicalFunction struct {
	Capacity int            `yaml:"capacity"`
	Vfs      []*PCIFunction `yaml:"vfs"`

	PCIFunction
}

// GetVirtualFunctions returns pcipf.Vfs
func (pcipf *PCIPhysicalFunction) GetVirtualFunctions() ([]pcifunction.BindablePCIFunction, error) {
	var vfs []pcifunction.BindablePCIFunction
	for _, vf := range pcipf.Vfs {
		vfs = append(vfs, vf)
	}
	return vfs, nil
}

// PCIFunction is a stub for pcifunction.BindableFunction
type PCIFunction struct {
	Addr       string `yaml:"addr"`
	IfName     string `yaml:"ifName"`
	IommuGroup uint   `yaml:"iommuGroup"`
	Driver     string `yaml:"driver"`
}

// GetPCIAddress returns pcif.Addr
func (pcif *PCIFunction) GetPCIAddress() string {
	return pcif.Addr
}

// GetNetInterfaceName returns pcif.IfName
func (pcif *PCIFunction) GetNetInterfaceName() (string, error) {
	return pcif.IfName, nil
}

// GetIommuGroupID returns pcif.IommuGroup
func (pcif *PCIFunction) GetIommuGroupID() (uint, error) {
	return pcif.IommuGroup, nil
}

// GetBoundDriver returns pcif.Driver
func (pcif *PCIFunction) GetBoundDriver() (string, error) {
	return pcif.Driver, nil
}

// BindDriver sets pcif.Driver = driver
func (pcif *PCIFunction) BindDriver(driver string) error {
	pcif.Driver = driver
	return nil
}
