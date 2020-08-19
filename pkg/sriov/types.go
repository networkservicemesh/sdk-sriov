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

package sriov

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/api/pcifunction"
)

// VirtualFunctionState is a virtual function state
type VirtualFunctionState string

// DriverType is a driver type that is bound to virtual function
type DriverType string

const (
	// UsedVirtualFunction is virtual function is use state
	UsedVirtualFunction VirtualFunctionState = "used"
	// FreeVirtualFunction is virtual function free state
	FreeVirtualFunction VirtualFunctionState = "free"

	// NoDriver is no driver type
	NoDriver DriverType = "no-driver"
	// KernelDriver is kernel driver type
	KernelDriver DriverType = "kernel"
	// VfioPCIDriver is vfio-pci driver type
	VfioPCIDriver DriverType = "vfio-pci"
)

// NetResourcePool provides contains information about net devices
type NetResourcePool struct {
	HostName          string
	PhysicalFunctions []*PhysicalFunction
	IommuGroups       map[uint]DriverType
	lock              sync.Mutex
}

// InitResourcePool configures devices, specified in provided config and initializes resource pool with that devices
func InitResourcePool(ctx context.Context, config *ResourceDomain, pciFactory pcifunction.ConfigurableFactory) (*NetResourcePool, error) {
	rp := &NetResourcePool{
		HostName:          config.HostName,
		PhysicalFunctions: nil,
		IommuGroups:       map[uint]DriverType{},
		lock:              sync.Mutex{},
	}

	for _, device := range config.PCIDevices {
		pf, err := pciFactory.NewConfigurablePhysicalFunction(device.PCIAddress)
		if err != nil {
			return nil, errors.Wrap(err, "invalid device provided")
		}

		vfCapacity, err := pf.GetVirtualFunctionsCapacity()
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to determine virtual functions capacity for device: %s", pf.GetPCIAddress())
		}

		pfIfaceName, err := pf.GetNetInterfaceName()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to determine net interface name for device %s", pf.GetPCIAddress())
		}

		pfKernelDriverName, err := pf.GetBoundDriver()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to determine kernel driver name for physical function: %s", pf.GetPCIAddress())
		}

		pfIommuGroup, err := pf.GetIommuGroupID()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to determine iommu group number for physical function: %s", pf.GetPCIAddress())
		}
		rp.IommuGroups[pfIommuGroup] = NoDriver

		physfun := &PhysicalFunction{
			PCIAddress:               pf.GetPCIAddress(),
			BoundDriver:              KernelDriver, // Kernel driver is bound by default
			KernelDriverName:         pfKernelDriverName,
			IommuGroup:               pfIommuGroup,
			NetInterfaceName:         pfIfaceName,
			TargetPCIAddress:         device.Target.PCIAddress,
			Capability:               device.Capability,
			VirtualFunctionsCapacity: vfCapacity,
			VirtualFunctions:         map[*VirtualFunction]VirtualFunctionState{},
		}

		err = pf.CreateVirtualFunctions(physfun.VirtualFunctionsCapacity)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create virtual functions for device %s", pf.GetPCIAddress())
		}

		vfs, err := pf.GetVirtualFunctions()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to discover virtual functions for device %s", pf.GetPCIAddress())
		}

		for _, vf := range vfs {
			vfIfaceName, err := vf.GetNetInterfaceName()
			if err != nil {
				return nil, errors.Wrapf(err, "unable to determine net interface name for device %s", vf.GetPCIAddress())
			}

			vfKernelDriverName, err := vf.GetBoundDriver()
			if err != nil {
				return nil, errors.Wrapf(err, "unable to kernel driver name for virtual function: %s", vf.GetPCIAddress())
			}

			vfIommuGroup, err := vf.GetIommuGroupID()
			if err != nil {
				return nil, errors.Wrapf(err, "unable to determine iommu group number for virtual function: %s", vf.GetPCIAddress())
			}
			rp.IommuGroups[vfIommuGroup] = NoDriver

			vf := &VirtualFunction{
				PCIAddress:       vf.GetPCIAddress(),
				BoundDriver:      KernelDriver, // Kernel driver is bound by default
				KernelDriverName: vfKernelDriverName,
				IommuGroup:       vfIommuGroup,
				NetInterfaceName: vfIfaceName,
			}

			physfun.VirtualFunctions[vf] = FreeVirtualFunction
		}

		rp.PhysicalFunctions = append(rp.PhysicalFunctions, physfun)
	}
	return rp, nil
}

// SelectVirtualFunction marks one of the free virtual functions for specified physical function as in-use and returns it
func (n *NetResourcePool) SelectVirtualFunction(pfPCIAddr string) (selectedVf *VirtualFunction, err error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, pf := range n.PhysicalFunctions {
		if pf.PCIAddress != pfPCIAddr {
			continue
		}

		// select the first free virtual function
		for vf, state := range pf.VirtualFunctions {
			if state == FreeVirtualFunction {
				selectedVf = vf
				break
			}
		}
		if selectedVf == nil {
			return nil, errors.Errorf("no free virtual function found for device %s", pfPCIAddr)
		}

		// mark it as in use
		err = pf.SetVirtualFunctionState(selectedVf, UsedVirtualFunction)
		if err != nil {
			return nil, err
		}

		return selectedVf, nil
	}

	return nil, errors.Errorf("no physical function with PCI address %s found", pfPCIAddr)
}

// ReleaseVirtualFunction marks given virtual function as free
func (n *NetResourcePool) ReleaseVirtualFunction(pfPCIAddr, vfNetIfaceName string) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, pf := range n.PhysicalFunctions {
		if pf.PCIAddress != pfPCIAddr {
			continue
		}

		for vf := range pf.VirtualFunctions {
			if vf.NetInterfaceName == vfNetIfaceName {
				return pf.SetVirtualFunctionState(vf, FreeVirtualFunction)
			}
		}
		return errors.Errorf("no virtual function with net interface name %s found", vfNetIfaceName)
	}
	return errors.Errorf("no physical function with PCI address %s found", pfPCIAddr)
}

// GetFreeVirtualFunctionsInfo returns map containing number of free virtual functions for each physical function
// in the pool keyed by physical function's PCI address
func (n *NetResourcePool) GetFreeVirtualFunctionsInfo() *FreeVirtualFunctionsInfo {
	n.lock.Lock()
	defer n.lock.Unlock()

	info := &FreeVirtualFunctionsInfo{
		HostName:             n.HostName,
		FreeVirtualFunctions: map[string]int{},
	}

	for _, pf := range n.PhysicalFunctions {
		freeVfs := pf.GetFreeVirtualFunctionsNumber()
		info.FreeVirtualFunctions[pf.PCIAddress] = freeVfs
	}

	return info
}

// PhysicalFunction contains information about physical function
type PhysicalFunction struct {
	PCIAddress               string
	BoundDriver              DriverType
	KernelDriverName         string
	IommuGroup               uint
	NetInterfaceName         string
	TargetPCIAddress         string
	Capability               string
	VirtualFunctionsCapacity uint
	VirtualFunctions         map[*VirtualFunction]VirtualFunctionState
}

// SetVirtualFunctionState changes state of the given virtual function
func (p *PhysicalFunction) SetVirtualFunctionState(vf *VirtualFunction, state VirtualFunctionState) error {
	val, found := p.VirtualFunctions[vf]
	if !found {
		return errors.New("specified virtual function is not found")
	}
	if val == state {
		return errors.Errorf("specified virtual function is already %s", state)
	}
	p.VirtualFunctions[vf] = state
	return nil
}

// GetFreeVirtualFunctionsNumber returns number of virtual functions that have FreeVirtualFunction state
func (p *PhysicalFunction) GetFreeVirtualFunctionsNumber() int {
	freeVfs := 0
	for _, state := range p.VirtualFunctions {
		if state == FreeVirtualFunction {
			freeVfs++
		}
	}
	return freeVfs
}

// VirtualFunction contains information about virtual function
type VirtualFunction struct {
	PCIAddress       string
	BoundDriver      DriverType
	KernelDriverName string
	IommuGroup       uint
	NetInterfaceName string
}
