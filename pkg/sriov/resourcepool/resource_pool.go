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

// Package resourcepool provides a resource pool for SR-IOV PCI virtual functions
package resourcepool

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/types/pcifunction"
	types "github.com/networkservicemesh/sdk-sriov/pkg/sriov/types/resourcepool"
)

// ResourcePool manages host SR-IOV state
// WARNING: it is thread unsafe - if you want to use it concurrently, use some synchronization outside
type ResourcePool struct {
	ctx               context.Context
	hostName          string
	iommuGroups       map[uint]*IommuGroup
	physicalFunctions map[string]*PhysicalFunction
	virtualFunctions  map[string]*VirtualFunction
}

// NewResourcePool returns a new ResourcePool
func NewResourcePool(ctx context.Context, pcifFactory pcifunction.Factory, config *Config) (*ResourcePool, error) {
	logEntry := log.Entry(ctx).WithField("ResourcePool", "NewResourcePool")

	rp := &ResourcePool{
		ctx:               ctx,
		hostName:          config.HostName,
		iommuGroups:       map[uint]*IommuGroup{},
		physicalFunctions: map[string]*PhysicalFunction{},
		virtualFunctions:  map[string]*VirtualFunction{},
	}

	for pfPciAddr, capability := range config.PhysicalFunctions {
		apiPf, err := pcifFactory.NewPhysicalFunction(pfPciAddr)
		if err != nil {
			logEntry.Errorf("invalid PF: %v", pfPciAddr)
			return nil, err
		}

		pf := &PhysicalFunction{
			Capability:       capability,
			VirtualFunctions: map[uint][]*VirtualFunction{},
		}

		apiVfs, err := apiPf.GetVirtualFunctions()
		if err != nil {
			logEntry.Errorf("failed to get VFs for the PF: %v", pfPciAddr)
			return nil, err
		}

		for _, apiVf := range apiVfs {
			vf := &VirtualFunction{
				Free: true,
			}

			if err := rp.addPCIFunction(&vf.PCIFunction, apiVf); err != nil {
				return nil, err
			}
			rp.virtualFunctions[vf.PCIAddress] = vf

			pf.VirtualFunctions[vf.IommuGroup.ID] = append(pf.VirtualFunctions[vf.IommuGroup.ID], vf)
		}

		if err := rp.addPCIFunction(&pf.PCIFunction, apiPf); err != nil {
			return nil, err
		}
		rp.physicalFunctions[pf.PCIAddress] = pf
	}

	return rp, nil
}

func (rp *ResourcePool) addPCIFunction(pcif *PCIFunction, apiPcif pcifunction.BindablePCIFunction) (err error) {
	logEntry := log.Entry(rp.ctx).WithField("ResourcePool", "addPCIFunction")

	pcif.PCIAddress = apiPcif.GetPCIAddress()
	pcif.DriverBinder = apiPcif

	if pcif.KernelDriverName, err = apiPcif.GetBoundDriver(); err != nil {
		logEntry.Errorf("failed to get driver name for the PCI function: %v", pcif.PCIAddress)
		return err
	}

	if pcif.NetInterfaceName, err = apiPcif.GetNetInterfaceName(); err != nil {
		logEntry.Errorf("failed to get network interface name for the PCI function: %v", pcif.PCIAddress)
		return err
	}

	igid, err := apiPcif.GetIommuGroupID()
	if err != nil {
		logEntry.Errorf("failed to get IOMMU group id for the PCI function: %v", pcif.PCIAddress)
		return err
	}

	ig, ok := rp.iommuGroups[igid]
	if !ok {
		ig = &IommuGroup{
			ID:           igid,
			BoundDriver:  types.NoDriver,
			PCIFunctions: []*PCIFunction{pcif},
		}
		rp.iommuGroups[igid] = ig
	} else {
		ig.PCIFunctions = append(ig.PCIFunctions, pcif)
	}
	pcif.IommuGroup = ig

	return nil
}

// GetHostInfo returns host SR-IOV state
func (rp *ResourcePool) GetHostInfo() *types.HostInfo {
	host := &types.HostInfo{
		HostName:          rp.hostName,
		PhysicalFunctions: make(map[string]*types.PhysicalFunctionInfo, len(rp.physicalFunctions)),
	}

	for pciAddr, pf := range rp.physicalFunctions {
		pfInfo := &types.PhysicalFunctionInfo{
			Capability:  pf.Capability,
			IommuGroups: make(map[uint]*types.IommuGroupInfo, len(pf.VirtualFunctions)),
		}
		for igid := range pf.VirtualFunctions {
			totalVfs, freeVfs := pf.GetVirtualFunctionsInfo(igid)
			pfInfo.IommuGroups[igid] = &types.IommuGroupInfo{
				DriverType:            rp.iommuGroups[igid].BoundDriver,
				TotalVirtualFunctions: totalVfs,
				FreeVirtualFunctions:  freeVfs,
			}
		}
		host.PhysicalFunctions[pciAddr] = pfInfo
	}

	return host
}

// Select selects a virtual function for the given physical function and IOMMU group,
// binds it to the given driver type and marks it as "in-use"
func (rp *ResourcePool) Select(pfPciAddr string, igid uint, driverType types.DriverType) (pcifunction.PCIFunction, error) {
	return rp.selectVF(pfPciAddr, func(pf *PhysicalFunction) (*VirtualFunction, error) {
		return pf.SelectVirtualFunction(igid)
	}, driverType)
}

// SelectAny selects a virtual function for the given physical function, binds it to the
// given driver type and marks it as "in-use"
func (rp *ResourcePool) SelectAny(pfPciAddr string, driverType types.DriverType) (pcifunction.PCIFunction, error) {
	return rp.selectVF(pfPciAddr, func(pf *PhysicalFunction) (*VirtualFunction, error) {
		return pf.SelectAnyVirtualFunction()
	}, driverType)
}

type vfSelector func(*PhysicalFunction) (*VirtualFunction, error)

func (rp *ResourcePool) selectVF(pfPciAddr string, vfSelect vfSelector, driverType types.DriverType) (*VirtualFunction, error) {
	pf, ok := rp.physicalFunctions[pfPciAddr]
	if !ok {
		return nil, errors.Errorf("trying to select for not existing PF PCI address = %v", pfPciAddr)
	}

	vf, err := vfSelect(pf)
	if err != nil {
		return nil, err
	}

	switch vf.IommuGroup.BoundDriver {
	case types.NoDriver:
		if err := rp.bindDriver(vf.IommuGroup, driverType); err != nil {
			return nil, err
		}
		vf.IommuGroup.BoundDriver = driverType
		fallthrough
	case driverType:
		vf.Free = false
	default:
		return nil, errors.Errorf("trying to rebind driver for the IOMMU group: %v", vf.IommuGroup.ID)
	}

	return vf, nil
}

// Free marks given virtual function as "free" and binds it to the "NoDriver" driver type
func (rp *ResourcePool) Free(vfPciAddr string) {
	logEntry := log.Entry(rp.ctx).WithField("ResourcePool", "Free")

	vf, ok := rp.virtualFunctions[vfPciAddr]
	if !ok {
		logEntry.Warnf("trying to free not existing VF: %v", vfPciAddr)
		return
	}

	vf.Free = true
	for _, pcif := range vf.IommuGroup.PCIFunctions {
		if igvf, ok := rp.virtualFunctions[pcif.PCIAddress]; ok && !igvf.Free {
			return
		}
	}

	vf.IommuGroup.BoundDriver = types.NoDriver
	if err := rp.bindDriver(vf.IommuGroup, types.NoDriver); err != nil {
		logEntry.Warnf("failed to unbound driver for %v: %+v", vfPciAddr, err)
	}
}

func (rp *ResourcePool) bindDriver(ig *IommuGroup, driverType types.DriverType) error {
	switch driverType {
	case types.NoDriver, types.KernelDriver:
		for _, pcif := range ig.PCIFunctions {
			if err := pcif.BindDriver(pcif.KernelDriverName); err != nil {
				return err
			}
		}
	case types.VfioPCIDriver:
		for _, pcif := range ig.PCIFunctions {
			if err := pcif.BindDriver(string(types.VfioPCIDriver)); err != nil {
				return err
			}
		}
	default:
		return errors.New("can only bind to the kernel or VFIO driver")
	}
	return nil
}
