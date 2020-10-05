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
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/sriovtest"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/yamlhelper"
)

const (
	physicalFunctionsFilename = "physical_functions.yml"
)

func initResourcePoolServer(driverType sriov.DriverType) (networkservice.NetworkServiceServer, []*sriovtest.PCIPhysicalFunction) {
	var pfs []*sriovtest.PCIPhysicalFunction
	_ = yamlhelper.UnmarshalFile(physicalFunctionsFilename, &pfs)

	functions := map[sriov.PCIFunction][]sriov.PCIFunction{}
	binders := map[uint][]sriov.DriverBinder{}
	for _, pf := range pfs {
		for _, vf := range pf.Vfs {
			functions[pf] = append(functions[pf], vf)
			binders[vf.IommuGroup] = append(binders[vf.IommuGroup], vf)
		}
	}

	return chain.NewNetworkServiceServer(resourcepool.NewServer(driverType, functions, binders)), pfs
}

func Test_resourcePoolServer_Request(t *testing.T) {
	vfConfig := &vfconfig.VFConfig{}
	ctx := vfconfig.WithConfig(context.TODO(), vfConfig)

	resourcePool := &resourcePoolMock{}
	ctx = resourcepool.WithPool(ctx, resourcePool)

	server, pfs := initResourcePoolServer(sriov.VfioPCIDriver)

	// 1. Request

	resourcePool.mock.On("Select", "1", sriov.VfioPCIDriver).
		Return(pfs[1].Vfs[1].Addr, nil)

	conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
			Mechanism: &networkservice.Mechanism{
				Type: vfio.MECHANISM,
				Parameters: map[string]string{
					resourcepool.TokenIDKey: "1",
				},
			},
		},
	})
	require.NoError(t, err)

	resourcePool.mock.AssertNumberOfCalls(t, "Select", 1)

	require.Equal(t, pfs[1].Vfs[0].Driver, string(sriov.VfioPCIDriver))
	require.Equal(t, pfs[1].Vfs[1].Driver, string(sriov.VfioPCIDriver))

	require.Equal(t, vfConfig.PFInterfaceName, pfs[1].IfName)
	require.Equal(t, vfConfig.VFInterfaceName, pfs[1].Vfs[1].IfName)
	require.Equal(t, vfConfig.VFNum, 1)

	require.Equal(t, vfio.ToMechanism(conn.Mechanism).GetIommuGroup(), pfs[1].Vfs[1].IommuGroup)

	// 2. Close

	resourcePool.mock.On("Free", pfs[1].Vfs[1].Addr).
		Return(nil)

	_, err = server.Close(ctx, conn)
	require.NoError(t, err)

	resourcePool.mock.AssertNumberOfCalls(t, "Free", 1)
}

type resourcePoolMock struct {
	mock mock.Mock

	sync.Mutex
}

func (rp *resourcePoolMock) Select(tokenID string, driverType sriov.DriverType) (string, error) {
	rv := rp.mock.Called(tokenID, driverType)
	return rv.String(0), rv.Error(1)
}

func (rp *resourcePoolMock) Free(vfPciAddr string) error {
	rv := rp.mock.Called(vfPciAddr)
	return rv.Error(0)
}
