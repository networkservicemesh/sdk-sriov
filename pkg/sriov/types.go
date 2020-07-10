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

// GetFreeVirtualFunctionsInfo returns map containing number of free virtual functions for each physical function
// in the pool keyed by physical function's PCI address
func (n *NetResourcePool) GetFreeVirtualFunctionsInfo() *FreeVirtualFunctionsInfo {
	n.lock.Lock()
	defer n.lock.Unlock()

	info := &FreeVirtualFunctionsInfo{
		FreeVirtualFunctions: map[string]int{},
	}

	for _, netResource := range n.Resources {
		pf := netResource.PhysicalFunction
		freeVfs := pf.GetFreeVirtualFunctionsNumber()
		info.FreeVirtualFunctions[pf.PCIAddress] = freeVfs
	}

	return info
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
	NetInterfaceName string
}
