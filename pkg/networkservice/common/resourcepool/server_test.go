// Copyright (c) 2022 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
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

//go:build linux
// +build linux

package resourcepool_test

import (
	"context"
	"sync"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/pci"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/sriovtest"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/yamlhelper"
)

const (
	physicalFunctionsFilename = "physical_functions.yml"
	configFileName            = "config.yml"
	pf2PciAddr                = "0000:00:02.0"
	vf2KernelDriver           = "vf-2-driver"
	tokenID                   = "sriov-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
)

type sample struct {
	driverType sriov.DriverType
	mechanism  string
	test       func(t *testing.T, pfs map[string]*sriovtest.PCIPhysicalFunction, vfConfig *vfconfig.VFConfig, conn *networkservice.Connection)
}

var samples = []*sample{
	{
		driverType: sriov.KernelDriver,
		mechanism:  kernel.MECHANISM,
		test: func(t *testing.T, pfs map[string]*sriovtest.PCIPhysicalFunction, vfConfig *vfconfig.VFConfig, _ *networkservice.Connection) {
			require.Equal(t, vf2KernelDriver, pfs[pf2PciAddr].Vfs[0].Driver)
			require.Equal(t, vf2KernelDriver, pfs[pf2PciAddr].Vfs[1].Driver)

			require.Equal(t, &vfconfig.VFConfig{
				PFInterfaceName: pfs[pf2PciAddr].IfName,
				VFInterfaceName: pfs[pf2PciAddr].Vfs[1].IfName,
				VFNum:           1,
			}, vfConfig)
		},
	},
	{
		driverType: sriov.VFIOPCIDriver,
		mechanism:  vfio.MECHANISM,
		test: func(t *testing.T, pfs map[string]*sriovtest.PCIPhysicalFunction, vfConfig *vfconfig.VFConfig, conn *networkservice.Connection) {
			require.Equal(t, string(sriov.VFIOPCIDriver), pfs[pf2PciAddr].Vfs[0].Driver)
			require.Equal(t, string(sriov.VFIOPCIDriver), pfs[pf2PciAddr].Vfs[1].Driver)

			require.Equal(t, &vfconfig.VFConfig{
				PFInterfaceName: pfs[pf2PciAddr].IfName,
				VFNum:           1,
			}, vfConfig)

			require.Equal(t, vfio.ToMechanism(conn.Mechanism).GetIommuGroup(), pfs[pf2PciAddr].Vfs[1].IOMMUGroup)
		},
	},
}

type vfResource struct {
	vfConfig *vfconfig.VFConfig
}

type vfResourceServer interface {
	networkservice.NetworkServiceServer
	getVFConfig() *vfconfig.VFConfig
}

func newVFResourceServer() vfResourceServer {
	return &vfResource{}
}

func (s *vfResource) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.vfConfig, _ = vfconfig.Load(ctx, false)
	return next.Server(ctx).Request(ctx, request)
}

func (s *vfResource) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (s *vfResource) getVFConfig() *vfconfig.VFConfig {
	return s.vfConfig
}

func TestResourcePoolServer_Request(t *testing.T) {
	for i := range samples {
		sample := samples[i]
		t.Run(sample.mechanism, func(t *testing.T) {
			var pfs map[string]*sriovtest.PCIPhysicalFunction
			_ = yamlhelper.UnmarshalFile(physicalFunctionsFilename, &pfs)

			conf, err := config.ReadConfig(context.TODO(), configFileName)
			require.NoError(t, err)

			pciPool, err := pci.NewTestPool(pfs, conf)
			require.NoError(t, err)

			resourcePool := new(resourcePoolMock)
			resourceServerChainElem := newVFResourceServer()

			server := chain.NewNetworkServiceServer(
				metadata.NewServer(),
				resourcepool.NewServer(sample.driverType, new(sync.Mutex), pciPool, resourcePool, conf),
				resourceServerChainElem)

			// 1. Request

			resourcePool.mock.On("Select", tokenID, sample.driverType).
				Return(pfs[pf2PciAddr].Vfs[1].Addr, nil)

			ctx := context.TODO()
			conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "id",
					Mechanism: &networkservice.Mechanism{
						Type: sample.mechanism,
						Parameters: map[string]string{
							common.DeviceTokenIDKey: tokenID,
						},
					},
				},
			})
			require.NoError(t, err)

			resourcePool.mock.AssertNumberOfCalls(t, "Select", 1)
			sample.test(t, pfs, resourceServerChainElem.getVFConfig(), conn)

			// 2. Close

			resourcePool.mock.On("Free", pfs[pf2PciAddr].Vfs[1].Addr).
				Return(nil)

			_, err = server.Close(context.TODO(), conn)
			require.NoError(t, err)

			resourcePool.mock.AssertNumberOfCalls(t, "Free", 1)
		})
	}
}

type resourcePoolMock struct {
	mock mock.Mock

	sync.Mutex
}

func (rp *resourcePoolMock) Select(tokenID string, driverType sriov.DriverType) (string, error) {
	rv := rp.mock.Called(tokenID, driverType)
	return rv.String(0), rv.Error(1)
}

func (rp *resourcePoolMock) Free(vfPCIAddr string) error {
	rv := rp.mock.Called(vfPCIAddr)
	return rv.Error(0)
}
