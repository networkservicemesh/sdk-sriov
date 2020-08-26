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
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/yamlhelper"
	"github.com/networkservicemesh/sdk-sriov/test/sriovtest"
)

const (
	physicalFunctionsFilename = "physical_functions.yml"
	hostInfoFileName          = "host_info.yml"
	vf11PciAddr               = "0000:01:00.1"
)

func testHostInfo() *sriov.HostInfo {
	host := &sriov.HostInfo{}
	_ = yamlhelper.UnmarshalFile(hostInfoFileName, host)
	return host
}

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

	return resourcepool.NewResourcePool(context.TODO(), vfs, config)
}

func TestResourcePool_GetHostInfo(t *testing.T) {
	rp := initResourcePool(t)

	assert.Equal(t, testHostInfo(), rp.GetHostInfo())
}

func TestResourcePool_Select(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.Select(pf1PciAddr, 2, sriov.VfioPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)

	host := testHostInfo()
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].DriverType = sriov.VfioPCIDriver
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].FreeVirtualFunctions = 0
	host.PhysicalFunctions[pf2PciAddr].IommuGroups[2].DriverType = sriov.VfioPCIDriver
	host.PhysicalFunctions[pf3PciAddr].IommuGroups[2].DriverType = sriov.VfioPCIDriver
	assert.Equal(t, host, rp.GetHostInfo())
}

func TestResourcePool_SelectAny(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.SelectAny(pf1PciAddr, sriov.VfioPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPciAddr)

	host := testHostInfo()
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].DriverType = sriov.VfioPCIDriver
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].FreeVirtualFunctions = 0
	host.PhysicalFunctions[pf2PciAddr].IommuGroups[2].DriverType = sriov.VfioPCIDriver
	host.PhysicalFunctions[pf3PciAddr].IommuGroups[2].DriverType = sriov.VfioPCIDriver
	assert.Equal(t, host, rp.GetHostInfo())
}

func TestResourcePool_Free(t *testing.T) {
	rp := initResourcePool(t)

	vfPciAddr, err := rp.SelectAny(pf1PciAddr, sriov.VfioPCIDriver)
	assert.Nil(t, err)
	rp.Free(vfPciAddr)

	assert.Equal(t, testHostInfo(), rp.GetHostInfo())
}
