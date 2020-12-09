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

// Package pci provides utils to work with pcifunction.Function
package pci

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/pcifunction"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/sriovtest"
)

const (
	vfioDriver = "vfio-pci"
)

type pciFunction interface {
	GetBoundDriver() (string, error)
	BindDriver(driver string) error

	sriov.PCIFunction
}

// Pool manages pcifunction.Function
type Pool struct {
	functions             map[string]*function // pciAddr -> *function
	functionsByIOMMUGroup map[uint][]*function // iommuGroup -> []*function
}

type function struct {
	function     pciFunction
	kernelDriver string
}

// NewPool returns a new PCI Pool
func NewPool(pciDevicesPath, pciDriversPath string, cfg *config.Config) (*Pool, error) {
	p := &Pool{
		functions:             map[string]*function{},
		functionsByIOMMUGroup: map[uint][]*function{},
	}

	for pfPCIAddr, pfCfg := range cfg.PhysicalFunctions {
		pf, err := pcifunction.NewPhysicalFunction(pfPCIAddr, pciDevicesPath, pciDriversPath)
		if err != nil {
			return nil, err
		}

		if err := p.addFunction(&pf.Function, pfCfg.PFKernelDriver); err != nil {
			return nil, err
		}

		for _, vf := range pf.GetVirtualFunctions() {
			if err := p.addFunction(vf, pfCfg.VFKernelDriver); err != nil {
				return nil, err
			}
		}
	}

	return p, nil
}

// NewTestPool returns a new PCI Pool for testing
func NewTestPool(physicalFunctions map[string]*sriovtest.PCIPhysicalFunction, cfg *config.Config) (*Pool, error) {
	p := &Pool{
		functions:             map[string]*function{},
		functionsByIOMMUGroup: map[uint][]*function{},
	}

	for pfPCIAddr, pfCfg := range cfg.PhysicalFunctions {
		pf, ok := physicalFunctions[pfPCIAddr]
		if !ok {
			return nil, errors.Errorf("PF doesn't exist: %v", pfPCIAddr)
		}

		_ = p.addFunction(&pf.PCIFunction, pfCfg.PFKernelDriver)

		for _, vf := range pf.Vfs {
			_ = p.addFunction(vf, pfCfg.VFKernelDriver)
		}
	}

	return p, nil
}

func (p *Pool) addFunction(pcif pciFunction, kernelDriver string) (err error) {
	f := &function{
		function:     pcif,
		kernelDriver: kernelDriver,
	}

	p.functions[pcif.GetPCIAddress()] = f

	iommuGroup, err := pcif.GetIOMMUGroup()
	if err != nil {
		return err
	}
	p.functionsByIOMMUGroup[iommuGroup] = append(p.functionsByIOMMUGroup[iommuGroup], f)

	return nil
}

// GetPCIFunction returns PCI function for the given PCI address
func (p *Pool) GetPCIFunction(pciAddr string) (sriov.PCIFunction, error) {
	f, err := p.find(pciAddr)
	if err != nil {
		return nil, err
	}
	return f.function, nil
}

// BindDriver binds selected IOMMU group to the given driver type
func (p *Pool) BindDriver(iommuGroup uint, driverType sriov.DriverType) error {
	for _, f := range p.functionsByIOMMUGroup[iommuGroup] {
		switch driverType {
		case sriov.KernelDriver:
			if err := f.function.BindDriver(f.kernelDriver); err != nil {
				return err
			}
		case sriov.VFIOPCIDriver:
			if err := f.function.BindDriver(vfioDriver); err != nil {
				return err
			}
		default:
			return errors.Errorf("driver type is not supported: %v", driverType)
		}
	}
	return nil
}

func (p *Pool) find(pciAddr string) (*function, error) {
	f, ok := p.functions[pciAddr]
	if !ok {
		return nil, errors.Errorf("PCI function doesn't exist: %v", pciAddr)
	}
	return f, nil
}
