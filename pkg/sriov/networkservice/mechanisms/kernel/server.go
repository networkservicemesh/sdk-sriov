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

// Package kernel provides a networkservice chain element that properly handles the SR-IOV kernel Mechanism
package kernel

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/utils"
)

type kernelServer struct {
	resourcePool *sriov.NetResourcePool
}

// NewServer - returns a new authorization networkservicemesh.NetworkServiceServers
func NewServer(resourcePool *sriov.NetResourcePool) networkservice.NetworkServiceServer {
	return &kernelServer{
		resourcePool: resourcePool,
	}
}

func (a *kernelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection() == nil {
		request.Connection = &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				ExtraContext: map[string]string{},
			},
		}
	}
	err := utils.WithVirtualFunctionsState(ctx, request.GetConnection().GetContext(), a.resourcePool)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to write virtual functions state into the connection context")
	}
	return next.Server(ctx).Request(ctx, request)
}

func (a *kernelServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
