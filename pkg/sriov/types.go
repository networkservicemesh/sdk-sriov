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
	"sync"

	"github.com/pkg/errors"
)

// VirtualFunctionState is a virtual function state
type VirtualFunctionState string

const (
	// UsedVirtualFunction is virtual function is use state
	UsedVirtualFunction VirtualFunctionState = "used"
	// FreeVirtualFunction is virtual function free state
	FreeVirtualFunction VirtualFunctionState = "free"
)

// NetResourcePool provides contains information about net devices
type NetResourcePool struct {
	Resources []*NetResource
	lock      sync.Mutex
}

// SelectVirtualFunction marks one of the free virtual functions for specified physical function as in-use and returns it
func (n *NetResourcePool) SelectVirtualFunction(pfPCIAddr string) (selectedVf *VirtualFunction, err error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, netResource := range n.Resources {
		pf := netResource.PhysicalFunction
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

	for _, netResource := range n.Resources {
		pf := netResource.PhysicalFunction
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

// NetResource contains information about net device
type NetResource struct {
	Capability       string
	PhysicalFunction *PhysicalFunction
}

// PhysicalFunction contains information about physical function
type PhysicalFunction struct {
	PCIAddress               string
	VirtualFunctionsCapacity int
	NetInterfaceName         string
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

// VirtualFunction contains information about virtual function
type VirtualFunction struct {
	PCIAddress       string
	NetInterfaceName string
}
