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

// Package types contains common type interfaces
package types

import "github.com/networkservicemesh/sdk-sriov/pkg/sriov"

// NetResourcePool provides an interface for accessing net device resources
type NetResourcePool interface {
	AddNetDevices(cfg *sriov.Config) error
	GetResources() []NetResource
}

// NetResource contains information about concrete resource in net resource pool
type NetResource interface {
	GetCapability() string
	GetPhysicalFunction() PhysicalFunction
}

// PhysicalFunction provides an interface to get information about physical function
type PhysicalFunction interface {
	GetPCIAddress() string
	GetVirtualFunctionsCapacity() int
	GetVirtualFunctionsInUse() []VirtualFunction
	GetFreeVirtualFunctions() []VirtualFunction
	SetVirtualFunctionInUse(VirtualFunction)
	SetVirtualFunctionFree(VirtualFunction)
	GetNetInterfaceName() string
}

// VirtualFunction provides an interface to get information about virtual function
type VirtualFunction interface {
	GetPCIAddress() string
	GetNetInterfaceName() string
}
