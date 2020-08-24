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

// Package hostinfo contains chain element for adding to Connection info about host SR-IOV state
package hostinfo

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/types/resourcepool"
)

type hostInfoServer struct {
	hostInfoProvider resourcepool.HostInfoProvider
}

// NewServer - returns a new networkservicemesh.NetworkServiceServer for adding host info
func NewServer(hostInfoProvider resourcepool.HostInfoProvider) networkservice.NetworkServiceServer {
	return &hostInfoServer{
		hostInfoProvider: hostInfoProvider,
	}
}

func (a *hostInfoServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := a.addHostInfo(ctx, request.GetConnection())
	if err != nil {
		return nil, errors.Wrap(err, "Unable to write host info into the connection context")
	}
	request.Connection = conn
	return next.Server(ctx).Request(ctx, request)
}

func (a *hostInfoServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (a *hostInfoServer) addHostInfo(ctx context.Context, conn *networkservice.Connection) (*networkservice.Connection, error) {
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetExtraContext() == nil {
		conn.Context.ExtraContext = map[string]string{}
	}

	info := a.hostInfoProvider.GetHostInfo()
	yamlInfo, err := yaml.Marshal(info)
	if err != nil {
		return nil, err
	}

	conn.GetContext().GetExtraContext()[resourcepool.HostInfoKey] = string(yamlInfo)
	log.Entry(ctx).Infof("Added info about free virtual functions into the ExtraContext for connection %s: %s", conn.GetId(), yamlInfo)

	return conn, nil
}
