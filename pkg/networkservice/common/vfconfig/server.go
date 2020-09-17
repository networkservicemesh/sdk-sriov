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

// Package vfconfig provides vfconfig chain element
package vfconfig

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vfConfigServer struct {
	configs map[string]*vfconfig.VFConfig
}

// NewServer returns a new vfconfig server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &vfConfigServer{
		configs: map[string]*vfconfig.VFConfig{},
	}
}

func (s *vfConfigServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	config, ok := s.configs[request.GetConnection().GetId()]
	if !ok {
		config = &vfconfig.VFConfig{}
		s.configs[request.GetConnection().GetId()] = config
	}

	return next.Server(ctx).Request(vfconfig.WithConfig(ctx, config), request)
}

func (s *vfConfigServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	config := s.configs[conn.GetId()]
	delete(s.configs, conn.GetId())

	return next.Server(ctx).Close(vfconfig.WithConfig(ctx, config), conn)
}
