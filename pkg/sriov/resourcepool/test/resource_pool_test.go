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

package resourcepool_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/sriovtest"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/yamlhelper"
)

const (
	physicalFunctionsFilename = "physical_functions.yml"
	vf11PciAddr               = "0000:01:00.1"
	vf22PciAddr               = "0000:02:00.2"
	vf31PciAddr               = "0000:03:00.1"
	vf51PciAddr               = "0000:05:00.1"
)

func initResourcePool(t *testing.T) *resourcepool.ResourcePool {
	var pfs []*sriovtest.PCIPhysicalFunction
	_ = yamlhelper.UnmarshalFile(physicalFunctionsFilename, &pfs)

	var vfs []*resourcepool.VirtualFunction
	for _, pf := range pfs {
		for _, vf := range pf.Vfs {
			vfs = append(vfs, &resourcepool.VirtualFunction{
				PCIAddress:                 vf.Addr,
				PhysicalFunctionPCIAddress: pf.Addr,
				IommuGroupID:               vf.IommuGroup,
			})
		}
	}

	config, err := resourcepool.ReadConfig(context.TODO(), configFileName)
	assert.Nil(t, err)

	return resourcepool.NewResourcePool(vfs, config)
}

func TestResourcePool_Select_Capability(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.Select(sriov.VfioPCIDriver, service1, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)

	vfPciAddr, err = rp.Select(sriov.VfioPCIDriver, service1, pf3Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf31PciAddr, vfPciAddr)
}

func TestResourcePool_Select_Service(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.Select(sriov.VfioPCIDriver, service1, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)

	vfPciAddr, err = rp.Select(sriov.VfioPCIDriver, service2, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf31PciAddr, vfPciAddr)
}

func TestResourcePool_Select_DriverType(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.Select(sriov.VfioPCIDriver, service1, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)

	vfPciAddr, err = rp.Select(sriov.KernelDriver, service1, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf22PciAddr, vfPciAddr)
}

func TestResourcePool_Select_FreeVirtualFunctionsCount(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.Select(sriov.VfioPCIDriver, service1, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)

	vfPciAddr, err = rp.Select(sriov.VfioPCIDriver, service1, pf4Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf51PciAddr, vfPciAddr)
}

func TestResourcePool_Free(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.Select(sriov.VfioPCIDriver, service1, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)

	err = rp.Free(vfPciAddr)
	assert.Nil(t, err)

	vfPciAddr, err = rp.Select(sriov.VfioPCIDriver, service1, pf1Capability)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)
}
