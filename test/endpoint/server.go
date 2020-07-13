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

// Package endpoint define a test endpoint listening on passed URL.
package endpoint

import (
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/common/selectorpciaddress"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type nseImpl struct {
	endpoint.Endpoint
}

// NewServer a new endpoint and running on grpc server
func NewServer(name string, authzServer networkservice.NetworkServiceServer, tokenGenerator token.GeneratorFunc, clientURL *url.URL, config *sriov.Config) endpoint.Endpoint {
	rv := &nseImpl{}
	rv.Endpoint = endpoint.NewServer(
		name,
		authzServer,
		tokenGenerator,
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernel.MECHANISM: selectorpciaddress.NewServer(config, kernel.PCIAddress),
			vfio.MECHANISM:   selectorpciaddress.NewServer(config, vfio.PCIAddress),
		}),
	)

	return rv
}
