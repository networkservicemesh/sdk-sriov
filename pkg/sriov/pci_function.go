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

// Package sriov provides types for SR-IOV
package sriov

// BindablePCIFunction is a PCIFunction and a DriverBinder
type BindablePCIFunction interface {
	PCIFunction
	DriverBinder
}

// PCIFunction provides methods to get OS PCI function info
type PCIFunction interface {
	GetPCIAddress() string
	GetNetInterfaceName() (string, error)
	GetIOMMUGroup() (uint, error)
	GetBoundDriver() (string, error)
}

// DriverBinder provides a method to bind driver to OS PCI function
type DriverBinder interface {
	BindDriver(driver string) error
	BindKernelDriver() error
}
