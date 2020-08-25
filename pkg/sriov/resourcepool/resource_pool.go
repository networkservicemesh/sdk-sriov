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

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

// ResourcePool manages host SR-IOV state
// WARNING: it is thread unsafe - if you want to use it concurrently, use some synchronization outside
type ResourcePool struct {
	ctx               context.Context
	hostName          string
	virtualFunctions  map[string]bool
	physicalFunctions map[string]*physicalFunction
	iommuGroups       map[uint]sriov.DriverType
}

type physicalFunction struct {
	capability       sriov.Capability
	virtualFunctions map[uint][]string
}

// NewResourcePool returns a new ResourcePool
func NewResourcePool(ctx context.Context, virtualFunctions []*VirtualFunction, config *Config) *ResourcePool {
	rp := &ResourcePool{
		ctx:               ctx,
		hostName:          config.HostName,
		virtualFunctions:  make(map[string]bool, len(virtualFunctions)),
		physicalFunctions: map[string]*physicalFunction{},
		iommuGroups:       map[uint]sriov.DriverType{},
	}

	for _, vf := range virtualFunctions {
		if capability, ok := config.PhysicalFunctions[vf.PhysicalFunctionPCIAddress]; ok {
			rp.virtualFunctions[vf.PCIAddress] = true

			if pf, ok := rp.physicalFunctions[vf.PhysicalFunctionPCIAddress]; !ok {
				rp.physicalFunctions[vf.PhysicalFunctionPCIAddress] = &physicalFunction{
					capability: capability,
					virtualFunctions: map[uint][]string{
						vf.IommuGroupID: {
							vf.PCIAddress,
						},
					},
				}
			} else {
				pf.virtualFunctions[vf.IommuGroupID] = append(pf.virtualFunctions[vf.IommuGroupID], vf.PCIAddress)
			}

			rp.iommuGroups[vf.IommuGroupID] = sriov.NoDriver
		}
	}

	return rp
}

// GetHostInfo returns host SR-IOV state
func (rp *ResourcePool) GetHostInfo() *sriov.HostInfo {
	host := &sriov.HostInfo{
		HostName:          rp.hostName,
		PhysicalFunctions: make(map[string]*sriov.PhysicalFunctionInfo, len(rp.physicalFunctions)),
	}

	for pciAddr, pf := range rp.physicalFunctions {
		pfInfo := &sriov.PhysicalFunctionInfo{
			Capability:  pf.capability,
			IommuGroups: make(map[uint]*sriov.IommuGroupInfo, len(pf.virtualFunctions)),
		}
		for igid, vfs := range pf.virtualFunctions {
			freeVfs := 0
			for _, vf := range vfs {
				if rp.virtualFunctions[vf] {
					freeVfs++
				}
			}
			pfInfo.IommuGroups[igid] = &sriov.IommuGroupInfo{
				DriverType:            rp.iommuGroups[igid],
				TotalVirtualFunctions: len(vfs),
				FreeVirtualFunctions:  freeVfs,
			}
		}
		host.PhysicalFunctions[pciAddr] = pfInfo
	}

	return host
}

// Select selects a virtual function for the given physical function and IOMMU group,
// binds it to the given driver type and marks it as "in-use"
func (rp *ResourcePool) Select(pfPciAddr string, igid uint, driverType sriov.DriverType) (string, error) {
	return rp.selectVF(pfPciAddr, func(pf *physicalFunction) (*VirtualFunction, error) {
		boundDriver := rp.iommuGroups[igid]
		if boundDriver == sriov.NoDriver || boundDriver == driverType {
			for _, vf := range pf.virtualFunctions[igid] {
				if rp.virtualFunctions[vf] {
					return &VirtualFunction{
						PCIAddress:   vf,
						IommuGroupID: igid,
					}, nil
				}
			}
		}
		return nil, errors.Errorf("no free VF for the PF, IOMMU group: %v, %v", pfPciAddr, igid)
	}, driverType)
}

// SelectAny selects a virtual function for the given physical function, binds it to the
// given driver type and marks it as "in-use"
func (rp *ResourcePool) SelectAny(pfPciAddr string, driverType sriov.DriverType) (string, error) {
	return rp.selectVF(pfPciAddr, func(pf *physicalFunction) (*VirtualFunction, error) {
		for igid, vfs := range pf.virtualFunctions {
			boundDriver := rp.iommuGroups[igid]
			if boundDriver == sriov.NoDriver || boundDriver == driverType {
				for _, vf := range vfs {
					if rp.virtualFunctions[vf] {
						return &VirtualFunction{
							PCIAddress:   vf,
							IommuGroupID: igid,
						}, nil
					}
				}
			}
		}
		return nil, errors.Errorf("no free VF for the PF: %v", pfPciAddr)
	}, driverType)
}

type vfSelector func(*physicalFunction) (*VirtualFunction, error)

func (rp *ResourcePool) selectVF(pfPciAddr string, vfSelect vfSelector, driverType sriov.DriverType) (string, error) {
	pf, ok := rp.physicalFunctions[pfPciAddr]
	if !ok {
		return "", errors.Errorf("trying to select for not existing PF PCI address = %v", pfPciAddr)
	}

	vf, err := vfSelect(pf)
	if err != nil {
		return "", err
	}

	switch rp.iommuGroups[vf.IommuGroupID] {
	case sriov.NoDriver:
		rp.iommuGroups[vf.IommuGroupID] = driverType
		fallthrough
	case driverType:
		rp.virtualFunctions[vf.PCIAddress] = false
	default:
		return "", errors.Errorf("trying to rebind driver for the IOMMU group: %v", vf.IommuGroupID)
	}

	return vf.PCIAddress, nil
}

// Free marks given virtual function as "free" and binds it to the "NoDriver" driver type
func (rp *ResourcePool) Free(vfPciAddr string) {
	logEntry := log.Entry(rp.ctx).WithField("ResourcePool", "Free")

	vf := rp.vfByPciAddr(vfPciAddr)
	if vf == (*VirtualFunction)(nil) {
		logEntry.Warnf("VF doesn't exist: %v", vfPciAddr)
		return
	}

	if rp.virtualFunctions[vf.PCIAddress] {
		logEntry.Warnf("trying to free not selected VF: %v", vf.PCIAddress)
		return
	}
	rp.virtualFunctions[vf.PCIAddress] = true

	for _, pf := range rp.physicalFunctions {
		if vfAddrs, ok := pf.virtualFunctions[vf.IommuGroupID]; ok {
			for _, vfAddr := range vfAddrs {
				if !rp.virtualFunctions[vfAddr] {
					return
				}
			}
		}
	}

	rp.iommuGroups[vf.IommuGroupID] = sriov.NoDriver
}

func (rp *ResourcePool) vfByPciAddr(vfPciAddr string) *VirtualFunction {
	for pfPciAddr, pf := range rp.physicalFunctions {
		for igid, vfAddrs := range pf.virtualFunctions {
			for _, vfAddr := range vfAddrs {
				if vfAddr == vfPciAddr {
					return &VirtualFunction{
						PCIAddress:                 vfPciAddr,
						PhysicalFunctionPCIAddress: pfPciAddr,
						IommuGroupID:               igid,
					}
				}
			}
		}
	}
	return nil
}
