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

// TODO: think about moving HostInfo to the api repository

// HostInfoKey is a key for HostInfo in context parameters
const HostInfoKey string = "HostInfo"

// HostInfo contains info about host SR-IOV state
type HostInfo struct {
	HostName          string                           `yaml:"hostname"`
	PhysicalFunctions map[string]*PhysicalFunctionInfo `yaml:"physicalFunctions"`
}

// PhysicalFunctionInfo contains info about physical function
type PhysicalFunctionInfo struct {
	Capability  Capability               `yaml:"capability"`
	IommuGroups map[uint]*IommuGroupInfo `yaml:"iommuGroups"`
}

// IommuGroupInfo contains info about virtual functions in single IOMMU group
// for the selected physical function
type IommuGroupInfo struct {
	DriverType            DriverType `yaml:"driverType"`
	TotalVirtualFunctions int        `yaml:"totalVirtualFunctions"`
	FreeVirtualFunctions  int        `yaml:"freeVirtualFunctions"`
}
