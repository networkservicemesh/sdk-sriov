// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/pcifunction"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/sriovtest"
)

const (
	vfioDriver        = "vfio-pci"
	driverBindTimeout = time.Second
	driverBindCheck   = driverBindTimeout / 10
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
	vfioDir               string
	skipDriverCheck       bool
}

type function struct {
	function     pciFunction
	kernelDriver string
}

// NewPool returns a new PCI Pool
func NewPool(pciDevicesPath, pciDriversPath, vfioDir string, cfg *config.Config) (*Pool, error) {
	return NewPCIPool(pciDevicesPath, pciDriversPath, vfioDir, cfg, false)
}

// NewPCIPool returns a new PCI Pool
func NewPCIPool(pciDevicesPath, pciDriversPath, vfioDir string, cfg *config.Config, skipDriverCheck bool) (*Pool, error) {
	p := &Pool{
		functions:             map[string]*function{},
		functionsByIOMMUGroup: map[uint][]*function{},
		vfioDir:               vfioDir,
		skipDriverCheck:       skipDriverCheck,
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
		skipDriverCheck:       true,
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
	f, ok := p.functions[pciAddr]
	if !ok {
		return nil, errors.Errorf("PCI function doesn't exist: %v", pciAddr)
	}
	return f.function, nil
}

// BindDriver binds selected IOMMU group to the given driver type
func (p *Pool) BindDriver(ctx context.Context, iommuGroup uint, driverType sriov.DriverType) error {
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

	for _, f := range p.functionsByIOMMUGroup[iommuGroup] {
		if err := p.waitDriverGettingBound(ctx, f.function, driverType); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pool) waitDriverGettingBound(ctx context.Context, pcif pciFunction, driverType sriov.DriverType) error {
	timeoutCh := time.After(driverBindTimeout)
	for {
		var driverCheck func(pciFunction) error
		switch driverType {
		case sriov.KernelDriver:
			driverCheck = p.kernelDriverCheck
		case sriov.VFIOPCIDriver:
			driverCheck = p.vfioDriverCheck
		default:
			return errors.Errorf("driver type is not supported: %v", driverType)
		}

		err := driverCheck(pcif)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "provided context is done")
		case <-timeoutCh:
			return errors.Errorf("time for binding kernel driver exceeded: %s, cause: %v", pcif.GetPCIAddress(), err)
		case <-time.After(driverBindCheck):
		}
	}
}

func (p *Pool) kernelDriverCheck(pcif pciFunction) error {
	if p.skipDriverCheck {
		return nil
	}

	_, err := pcif.GetNetInterfaceName()
	return err
}

func (p *Pool) vfioDriverCheck(pcif pciFunction) error {
	if p.skipDriverCheck {
		return nil
	}

	iommuGroup, err := pcif.GetIOMMUGroup()
	if err != nil {
		return err
	}

	_, err = os.Stat(filepath.Join(p.vfioDir, strconv.FormatUint(uint64(iommuGroup), 10)))
	return errors.Wrapf(err, "failed to join path elements: %s, %s", p.vfioDir, strconv.FormatUint(uint64(iommuGroup), 10))
}
