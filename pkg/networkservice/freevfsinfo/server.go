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

// Package freevfsinfo contains chain element for adding to Connection info about free virtual functions on the host
package freevfsinfo

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

type freeVirtualFunctionsInfoServer struct {
	resourcePool *sriov.NetResourcePool
}

// NewServer - returns a new networkservicemesh.NetworkServiceServer for adding free virtual functions info
func NewServer(resourcePool *sriov.NetResourcePool) networkservice.NetworkServiceServer {
	return &freeVirtualFunctionsInfoServer{
		resourcePool: resourcePool,
	}
}

func (a *freeVirtualFunctionsInfoServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := a.addFreeVirtualFunctionsInfo(ctx, request.GetConnection())
	if err != nil {
		return nil, errors.Wrap(err, "Unable to write info about free virtual functions into the connection context")
	}
	request.Connection = conn
	return next.Server(ctx).Request(ctx, request)
}

func (a *freeVirtualFunctionsInfoServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (a *freeVirtualFunctionsInfoServer) addFreeVirtualFunctionsInfo(ctx context.Context, conn *networkservice.Connection) (*networkservice.Connection, error) {
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetExtraContext() == nil {
		conn.Context.ExtraContext = map[string]string{}
	}

	info := a.resourcePool.GetFreeVirtualFunctionsInfo()
	yamlInfo, err := info.Marshall()
	if err != nil {
		return nil, err
	}

	conn.GetContext().GetExtraContext()[sriov.FreeVirtualFunctionsInfoKey] = yamlInfo
	log.Entry(ctx).Infof("Added info about free virtual functions into the ExtraContext for connection %s: %s", conn.GetId(), yamlInfo)

	return conn, nil
}
