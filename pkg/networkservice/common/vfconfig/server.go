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
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vfConfigServer struct {
	configs sync.Map
}

// NewServer returns a new vfconfig server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &vfConfigServer{
		configs: sync.Map{},
	}
}

func (s *vfConfigServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	rawConfig, _ := s.configs.LoadOrStore(request.GetConnection().GetId(), &vfconfig.VFConfig{})

	return next.Server(ctx).Request(vfconfig.WithConfig(ctx, rawConfig.(*vfconfig.VFConfig)), request)
}

func (s *vfConfigServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	rawConfig, ok := s.configs.Load(conn.GetId())
	if !ok {
		return nil, errors.Errorf("no VF config for the connection: %v", conn.GetId())
	}
	s.configs.Delete(conn.GetId())

	return next.Server(ctx).Close(vfconfig.WithConfig(ctx, rawConfig.(*vfconfig.VFConfig)), conn)
}
