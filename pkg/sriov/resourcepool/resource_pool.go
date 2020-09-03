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
	"sort"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

// ResourcePool manages host SR-IOV state
// WARNING: it is thread unsafe - if you want to use it concurrently, use some synchronization outside
type ResourcePool struct {
	virtualFunctions  map[string]bool
	physicalFunctions map[string]*physicalFunction
	iommuGroups       map[uint]sriov.DriverType
}

type physicalFunction struct {
	capability                sriov.Capability
	services                  map[string]bool
	virtualFunctions          map[uint][]string
	freeVirtualFunctionsCount int
}

func (pf *physicalFunction) compare(other *physicalFunction) int {
	if cmp := pf.capability.Compare(other.capability); cmp != 0 {
		return cmp
	}
	return other.freeVirtualFunctionsCount - pf.freeVirtualFunctionsCount
}

// NewResourcePool returns a new ResourcePool
func NewResourcePool(virtualFunctions []*VirtualFunction, config *Config) *ResourcePool {
	rp := &ResourcePool{
		virtualFunctions:  make(map[string]bool, len(virtualFunctions)),
		physicalFunctions: map[string]*physicalFunction{},
		iommuGroups:       map[uint]sriov.DriverType{},
	}

	for _, vf := range virtualFunctions {
		physFun, ok := config.PhysicalFunctions[vf.PhysicalFunctionPCIAddress]
		if !ok {
			continue
		}

		rp.virtualFunctions[vf.PCIAddress] = true

		pf, ok := rp.physicalFunctions[vf.PhysicalFunctionPCIAddress]
		if !ok {
			pf = &physicalFunction{
				capability:       physFun.Capability,
				services:         map[string]bool{},
				virtualFunctions: map[uint][]string{},
			}
			for _, service := range physFun.Services {
				pf.services[service] = true
			}
			rp.physicalFunctions[vf.PhysicalFunctionPCIAddress] = pf
		}
		pf.virtualFunctions[vf.IommuGroupID] = append(pf.virtualFunctions[vf.IommuGroupID], vf.PCIAddress)

		rp.iommuGroups[vf.IommuGroupID] = sriov.NoDriver
	}

	for _, pf := range rp.physicalFunctions {
		for _, vfs := range pf.virtualFunctions {
			pf.freeVirtualFunctionsCount += len(vfs)
		}
	}

	return rp
}

// Select selects a virtual function for the given driver type and marks it as "in-use"
func (rp *ResourcePool) Select(driverType sriov.DriverType, service string, capability sriov.Capability) (string, error) {
	vfs := rp.find(driverType, service, capability)
	if len(vfs) == 0 {
		return "", errors.Errorf("no free VF for the driver type: %v", driverType)
	}

	sort.Slice(vfs, func(i, k int) bool {
		iIg := rp.iommuGroups[vfs[i].IommuGroupID]
		kIg := rp.iommuGroups[vfs[k].IommuGroupID]
		iPf := rp.physicalFunctions[vfs[i].PhysicalFunctionPCIAddress]
		kPf := rp.physicalFunctions[vfs[k].PhysicalFunctionPCIAddress]
		switch {
		case iIg == driverType && kIg == sriov.NoDriver:
			return true
		case iIg == sriov.NoDriver && kIg == driverType:
			return false
		default:
			return iPf.compare(kPf) < 0
		}
	})
	vf := vfs[0]

	rp.virtualFunctions[vf.PCIAddress] = false
	rp.physicalFunctions[vf.PhysicalFunctionPCIAddress].freeVirtualFunctionsCount--
	rp.iommuGroups[vf.IommuGroupID] = driverType

	return vf.PCIAddress, nil
}

func (rp *ResourcePool) find(driverType sriov.DriverType, service string, capability sriov.Capability) []*VirtualFunction {
	var vfs []*VirtualFunction
	for pfPciAddr, pf := range rp.physicalFunctions {
		if pf.services[service] && pf.capability.Compare(capability) >= 0 {
			for igid, vfPciAddrs := range pf.virtualFunctions {
				if ig := rp.iommuGroups[igid]; ig == sriov.NoDriver || ig == driverType {
					for _, vfPciAddr := range vfPciAddrs {
						if rp.virtualFunctions[vfPciAddr] {
							vfs = append(vfs, &VirtualFunction{
								PCIAddress:                 vfPciAddr,
								PhysicalFunctionPCIAddress: pfPciAddr,
								IommuGroupID:               igid,
							})
						}
					}
				}
			}
		}
	}
	return vfs
}

// Free marks given virtual function as "free" and binds it to the "NoDriver" driver type
func (rp *ResourcePool) Free(vfPciAddr string) error {
	vf := rp.findByPciAddr(vfPciAddr)
	if vf == (*VirtualFunction)(nil) {
		return errors.Errorf("VF doesn't exist: %v", vfPciAddr)
	}

	if rp.virtualFunctions[vf.PCIAddress] {
		return errors.Errorf("trying to free not selected VF: %v", vf.PCIAddress)
	}

	rp.virtualFunctions[vf.PCIAddress] = true
	rp.physicalFunctions[vf.PhysicalFunctionPCIAddress].freeVirtualFunctionsCount++

	for _, pf := range rp.physicalFunctions {
		if vfAddrs, ok := pf.virtualFunctions[vf.IommuGroupID]; ok {
			for _, vfAddr := range vfAddrs {
				if !rp.virtualFunctions[vfAddr] {
					return nil
				}
			}
		}
	}
	rp.iommuGroups[vf.IommuGroupID] = sriov.NoDriver

	return nil
}

func (rp *ResourcePool) findByPciAddr(vfPciAddr string) *VirtualFunction {
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
