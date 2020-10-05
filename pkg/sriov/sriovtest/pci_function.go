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

package sriovtest

import "github.com/networkservicemesh/sdk-sriov/pkg/sriov"

// PCIPhysicalFunction is a test data class for pcifunction.PhysicalFunction
type PCIPhysicalFunction struct {
	Capacity int            `yaml:"capacity"`
	Vfs      []*PCIFunction `yaml:"vfs"`

	PCIFunction
}

// PCIFunction is a test data class for pcifunction.Function
type PCIFunction struct {
	Addr       string `yaml:"addr"`
	IfName     string `yaml:"ifName"`
	IOMMUGroup uint   `yaml:"iommuGroup"`
	Driver     string `yaml:"driver"`
}

// GetPCIAddress returns f.Addr
func (f *PCIFunction) GetPCIAddress() string {
	return f.Addr
}

// GetNetInterfaceName returns f.IfName
func (f *PCIFunction) GetNetInterfaceName() (string, error) {
	return f.IfName, nil
}

// GetIOMMUGroup returns f.IOMMUGroup
func (f *PCIFunction) GetIOMMUGroup() (uint, error) {
	return f.IOMMUGroup, nil
}

// GetBoundDriver returns f.Driver
func (f *PCIFunction) GetBoundDriver() (string, error) {
	return f.Driver, nil
}

// BindDriver sets f.Driver = driver
func (f *PCIFunction) BindDriver(driver string) error {
	f.Driver = driver
	return nil
}

// BindKernelDriver sets f.Driver = sriov.KernelDriver
func (f *PCIFunction) BindKernelDriver() error {
	f.Driver = string(sriov.KernelDriver)
	return nil
}
