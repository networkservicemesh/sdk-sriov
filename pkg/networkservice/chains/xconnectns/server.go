// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

//+build !windows

// Package xconnectns provides an Endpoint implementing the SR-IOV Forwarder networks service
package xconnectns

import (
	"context"
	"net/url"
	"sync"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	noopmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/ethernetcontext"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/inject"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/ipcontext"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/netns"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/rename"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/noop"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resetmechanism"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/vfconfig"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
)

type sriovServer struct {
	endpoint.Endpoint
}

// NewServer - returns an Endpoint implementing the SR-IOV Forwarder networks service
//             - name - name of the Forwarder
//             - authzServer - policy for allowing or rejecting requests
//             - tokenGenerator - token.GeneratorFunc - generates tokens for use in Path
//             - pciPool - provides PCI functions
//             - resourcePool - provides SR-IOV resources
//             - sriovConfig - SR-IOV PCI functions config
//             - vfioDir - host /dev/vfio directory mount location
//             - cgroupBaseDir - host /sys/fs/cgroup/devices directory mount location
//             - clientUrl - *url.URL for the talking to the NSMgr
//             - ...clientDialOptions - dialOptions for dialing the NSMgr
func NewServer(
	ctx context.Context,
	name string,
	authzServer networkservice.NetworkServiceServer,
	tokenGenerator token.GeneratorFunc,
	pciPool resourcepool.PCIPool,
	resourcePool resourcepool.ResourcePool,
	sriovConfig *config.Config,
	vfioDir, cgroupBaseDir string,
	clientURL *url.URL,
	clientDialOptions ...grpc.DialOption,
) endpoint.Endpoint {
	rv := sriovServer{}

	connectChainFactory := func(class string) networkservice.NetworkServiceServer {
		return chain.NewNetworkServiceServer(
			clienturl.NewServer(clientURL),
			heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
			connect.NewServer(ctx,
				client.NewCrossConnectClientFactory(
					client.WithName(name),
					client.WithAdditionalFunctionality(
						noop.NewClient(class),
					),
				),
				connect.WithDialOptions(clientDialOptions...),
			),
		)
	}

	resourceLock := &sync.Mutex{}
	sriovChain := chain.NewNetworkServiceServer(
		recvfd.NewServer(),
		vfconfig.NewServer(),
		resetmechanism.NewServer(
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				kernel.MECHANISM: chain.NewNetworkServiceServer(
					resourcepool.NewServer(sriov.KernelDriver, resourceLock, pciPool, resourcePool, sriovConfig),
					rename.NewServer(),
					inject.NewServer(),
				),
				vfiomech.MECHANISM: chain.NewNetworkServiceServer(
					resourcepool.NewServer(sriov.VFIOPCIDriver, resourceLock, pciPool, resourcePool, sriovConfig),
					vfio.NewServer(vfioDir, cgroupBaseDir),
				),
			}),
		),
		connectChainFactory(cls.REMOTE),
		// we setup VF ethernet context using PF interface, so we do it in the forwarder net NS
		ethernetcontext.NewVFServer(),
		// now setup VF interface, so we do it in the client net NS
		netns.NewServer(),
		rename.NewServer(),
		ipcontext.NewServer(),
	)

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAuthorizeServer(authzServer),
		endpoint.WithAdditionalFunctionality(
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				kernel.MECHANISM:   sriovChain,
				vfiomech.MECHANISM: sriovChain,
				noopmech.MECHANISM: connectChainFactory(cls.LOCAL),
			}),
		),
	)

	return rv
}
