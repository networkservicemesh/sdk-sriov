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

package utils

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

// WithVirtualFunctionsState adds information about free virtual functions number for each available physical function
// on host to the ExtraContext of the ConnectionContext
func WithVirtualFunctionsState(ctx context.Context, request *networkservice.NetworkServiceRequest, resourcePool *sriov.NetResourcePool) error {
	resourcePool.Lock()
	defer resourcePool.Unlock()

	if request.GetConnection() == nil {
		request.Connection = &networkservice.Connection{}
	}
	if request.GetConnection().GetContext() == nil {
		request.Connection.Context = &networkservice.ConnectionContext{}
	}
	if request.GetConnection().GetContext().GetExtraContext() == nil {
		request.Connection.Context.ExtraContext = map[string]string{}
	}

	config := &sriov.VirtualFunctionsStateConfig{
		Config: map[string]int{},
	}

	for _, netResource := range resourcePool.Resources {
		pf := netResource.PhysicalFunction
		freeVfs := getFreeVirtualFunctionsNumber(pf)

		config.Config[pf.PCIAddress] = freeVfs
	}

	strCfg, err := sriov.MarshallStateConfig(ctx, config)
	if err != nil {
		return err
	}

	request.GetConnection().GetContext().GetExtraContext()[sriov.VirtualFunctionsStateConfigKey] = strCfg
	return nil
}

func getFreeVirtualFunctionsNumber(pf *sriov.PhysicalFunction) int {
	freeVfs := 0
	for _, state := range pf.VirtualFunctions {
		if state == sriov.FreeVirtualFunction {
			freeVfs++
		}
	}
	return freeVfs
}
