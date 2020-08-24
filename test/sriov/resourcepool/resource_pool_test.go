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

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/types/pcifunction"
	types "github.com/networkservicemesh/sdk-sriov/pkg/sriov/types/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/yamlhelper"
	"github.com/networkservicemesh/sdk-sriov/test/stub"
)

const (
	pciFunctionFactoryFilename = "pci_function_factory.yml"
	hostInfoFileName           = "host_info.yml"
	vf11PciAddr                = "0000:01:00.1"
	vf11IfName                 = "vf-1-1-ifname"
	vf11Driver                 = "vf-1-1-driver"
	vf22Driver                 = "vf-2-2-driver"
	vf32Driver                 = "vf-3-2-driver"
)

func testPCIFunctionFactory() *stub.PCIFunctionFactory {
	pciff := &stub.PCIFunctionFactory{}
	_ = yamlhelper.UnmarshalFile(pciFunctionFactoryFilename, pciff)
	return pciff
}

func testHostInfo() *types.HostInfo {
	host := &types.HostInfo{}
	_ = yamlhelper.UnmarshalFile(hostInfoFileName, host)
	return host
}

func initResourcePool(t *testing.T) (rp *resourcepool.ResourcePool, pciff *stub.PCIFunctionFactory) {
	pciff = testPCIFunctionFactory()

	config, err := resourcepool.ReadConfig(context.TODO(), configFileName)
	assert.Nil(t, err)

	rp, err = resourcepool.NewResourcePool(context.TODO(), pciff, config)
	assert.Nil(t, err)

	return rp, pciff
}

func TestResourcePool_GetHostInfo(t *testing.T) {
	rp, _ := initResourcePool(t)

	assert.Equal(t, testHostInfo(), rp.GetHostInfo())
}

func TestResourcePool_Select(t *testing.T) {
	rp, pciff := initResourcePool(t)

	vf, err := rp.Select(pf1PciAddr, 2, types.VfioPCIDriver)
	assert.Nil(t, err)
	assertPCIFunctionEqual(t, &stub.PCIFunction{
		Addr:       vf11PciAddr,
		IfName:     vf11IfName,
		IommuGroup: 2,
		Driver:     string(types.VfioPCIDriver),
	}, vf)

	host := testHostInfo()
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].DriverType = types.VfioPCIDriver
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].FreeVirtualFunctions = 0
	host.PhysicalFunctions[pf2PciAddr].IommuGroups[2].DriverType = types.VfioPCIDriver
	host.PhysicalFunctions[pf3PciAddr].IommuGroups[2].DriverType = types.VfioPCIDriver
	assert.Equal(t, host, rp.GetHostInfo())

	assert.Equal(t, string(types.VfioPCIDriver), pciff.Pfs[0].Vfs[0].Driver)
	assert.Equal(t, string(types.VfioPCIDriver), pciff.Pfs[1].Vfs[1].Driver)
	assert.Equal(t, string(types.VfioPCIDriver), pciff.Pfs[2].Vfs[1].Driver)
}

func TestResourcePool_SelectAny(t *testing.T) {
	rp, pciff := initResourcePool(t)

	vf, err := rp.SelectAny(pf1PciAddr, types.VfioPCIDriver)
	assert.Nil(t, err)
	assertPCIFunctionEqual(t, &stub.PCIFunction{
		Addr:       vf11PciAddr,
		IfName:     vf11IfName,
		IommuGroup: 2,
		Driver:     string(types.VfioPCIDriver),
	}, vf)

	host := testHostInfo()
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].DriverType = types.VfioPCIDriver
	host.PhysicalFunctions[pf1PciAddr].IommuGroups[2].FreeVirtualFunctions = 0
	host.PhysicalFunctions[pf2PciAddr].IommuGroups[2].DriverType = types.VfioPCIDriver
	host.PhysicalFunctions[pf3PciAddr].IommuGroups[2].DriverType = types.VfioPCIDriver
	assert.Equal(t, host, rp.GetHostInfo())

	assert.Equal(t, string(types.VfioPCIDriver), pciff.Pfs[0].Vfs[0].Driver)
	assert.Equal(t, string(types.VfioPCIDriver), pciff.Pfs[1].Vfs[1].Driver)
	assert.Equal(t, string(types.VfioPCIDriver), pciff.Pfs[2].Vfs[1].Driver)
}

func assertPCIFunctionEqual(t *testing.T, expected, actual pcifunction.PCIFunction) {
	assert.Equal(t, expected.GetPCIAddress(), actual.GetPCIAddress())

	expectedIfName, _ := expected.GetNetInterfaceName()
	actualIfName, err := actual.GetNetInterfaceName()
	assert.Nil(t, err)
	assert.Equal(t, expectedIfName, actualIfName)

	expectedIgid, _ := expected.GetIommuGroupID()
	actualIgid, err := actual.GetIommuGroupID()
	assert.Nil(t, err)
	assert.Equal(t, expectedIgid, actualIgid)

	expectedDriver, _ := expected.GetBoundDriver()
	actualDriver, err := actual.GetBoundDriver()
	assert.Nil(t, err)
	assert.Equal(t, expectedDriver, actualDriver)
}

func TestResourcePool_Free(t *testing.T) {
	rp, pciff := initResourcePool(t)

	vf, err := rp.SelectAny(pf1PciAddr, types.VfioPCIDriver)
	assert.Nil(t, err)
	assert.NotNil(t, vf)
	rp.Free(vf.GetPCIAddress())

	assert.Equal(t, testHostInfo(), rp.GetHostInfo())

	assert.Equal(t, vf11Driver, pciff.Pfs[0].Vfs[0].Driver)
	assert.Equal(t, vf22Driver, pciff.Pfs[1].Vfs[1].Driver)
	assert.Equal(t, vf32Driver, pciff.Pfs[2].Vfs[1].Driver)
}
