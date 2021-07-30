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

package vfconfig

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type vfConfigClient struct {
	configs sync.Map
}

// NewClient returns a new vfconfig client chain element
func NewClient() networkservice.NetworkServiceClient {
	return &vfConfigClient{
		configs: sync.Map{},
	}
}

func (c *vfConfigClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	rawConfig, _ := c.configs.LoadOrStore(conn.GetId(), &vfconfig.VFConfig{})
	vfconfig.WithConfig(ctx, rawConfig.(*vfconfig.VFConfig))

	return conn, nil
}

func (c *vfConfigClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}

	rawConfig, ok := c.configs.Load(conn.GetId())
	if !ok {
		return nil, errors.Errorf("no VF config for the connection: %v", conn.GetId())
	}
	c.configs.Delete(conn.GetId())
	vfconfig.WithConfig(ctx, rawConfig.(*vfconfig.VFConfig))
	return rv, err
}
