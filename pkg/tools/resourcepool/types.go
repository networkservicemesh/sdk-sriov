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

package resourcepool

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/tools/api/pcifunction"
	api "github.com/networkservicemesh/sdk-sriov/pkg/tools/api/resourcepool"
)

// IommuGroup contains information about IOMMU group
type IommuGroup struct {
	ID           uint
	BoundDriver  api.DriverType
	PCIFunctions []*PCIFunction
}

// PhysicalFunction contains information about physical PCI function
type PhysicalFunction struct {
	Capability       api.Capability
	VirtualFunctions map[uint][]*VirtualFunction

	PCIFunction
}

// GetVirtualFunctionsInfo returns total and free VFs count for the given IOMMU group
func (pf *PhysicalFunction) GetVirtualFunctionsInfo(igid uint) (total, free int) {
	vfs := pf.VirtualFunctions[igid]
	for _, vf := range vfs {
		if vf.Free {
			free++
		}
	}
	return len(vfs), free
}

// SelectVirtualFunction finds a free VF with the given IOMMU group id
func (pf *PhysicalFunction) SelectVirtualFunction(igid uint) (*VirtualFunction, error) {
	for _, vf := range pf.VirtualFunctions[igid] {
		if vf.Free {
			return vf, nil
		}
	}
	return nil, errors.Errorf("no available VFs for %v-%v", pf.PCIAddress, igid)
}

// SelectAnyVirtualFunction finds a free VF
func (pf *PhysicalFunction) SelectAnyVirtualFunction() (*VirtualFunction, error) {
	for igid := range pf.VirtualFunctions {
		if vf, err := pf.SelectVirtualFunction(igid); err == nil {
			return vf, nil
		}
	}
	return nil, errors.Errorf("no available VFs for %v", pf.PCIAddress)
}

// VirtualFunction contains information about virtual PCI function
type VirtualFunction struct {
	Free bool

	PCIFunction
}

// PCIFunction contains common information about PCI function
type PCIFunction struct {
	PCIAddress       string
	KernelDriverName string
	NetInterfaceName string
	IommuGroup       *IommuGroup

	pcifunction.DriverBinder
}

// GetPCIAddress returns PCI function PCI address
func (pcif *PCIFunction) GetPCIAddress() string {
	return pcif.PCIAddress
}

// GetNetInterfaceName returns PCI function network interface name
func (pcif *PCIFunction) GetNetInterfaceName() (string, error) {
	return pcif.NetInterfaceName, nil
}

// GetIommuGroupID returns PCI function IOMMU group id
func (pcif *PCIFunction) GetIommuGroupID() (uint, error) {
	return pcif.IommuGroup.ID, nil
}

// GetBoundDriver returns PCI function bound driver
func (pcif *PCIFunction) GetBoundDriver() (string, error) {
	return string(pcif.IommuGroup.BoundDriver), nil
}

var _ pcifunction.PCIFunction = (*PCIFunction)(nil)
