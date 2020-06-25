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

// Package types contains common type structures
package types

import (
	"sync"

	"github.com/pkg/errors"
)

const (
	// VirtualFunctionInUse is virtual function is use state
	VirtualFunctionInUse = "inUse"
	// FreeVirtualFunction is virtual function free state
	FreeVirtualFunction = "free"
)

// NetResourcePool provides contains information about net devices
type NetResourcePool struct {
	Resources []*NetResource
	sync.Mutex
}

// NetResource contains information about net device
type NetResource struct {
	Capability       string
	PhysicalFunction PhysicalFunction
}

// PhysicalFunction contains information about physical function
type PhysicalFunction struct {
	PCIAddress               string
	VirtualFunctionsCapacity int
	NetInterfaceName         string
	VirtualFunctions         map[VirtualFunction]string
	sync.Mutex
}

// SetVirtualFunctionInUse moves given free virtual function into the virtual functions in use map
func (p *PhysicalFunction) SetVirtualFunctionInUse(vf VirtualFunction) error {
	return p.setVirtualFunctionState(vf, VirtualFunctionInUse)
}

// SetVirtualFunctionFree moves given virtual function in use into the free virtual functions map
func (p *PhysicalFunction) SetVirtualFunctionFree(vf VirtualFunction) error {
	return p.setVirtualFunctionState(vf, FreeVirtualFunction)
}

func (p *PhysicalFunction) setVirtualFunctionState(vf VirtualFunction, state string) error {
	p.Lock()
	defer p.Unlock()

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

// VirtualFunction provides contains information about virtual function
type VirtualFunction struct {
	PCIAddress       string
	NetInterfaceName string
}
