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

package resourcepool

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/resource"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/token"
)

type initResourcePoolServer struct {
	resourcePool *resource.Pool
	lock         sync.Mutex
}

// NewInitServer returns a new init resource pool server
func NewInitServer(tokenPool *token.Pool, cfg *config.Config) networkservice.NetworkServiceServer {
	return &initResourcePoolServer{
		resourcePool: resource.NewPool(tokenPool, cfg),
	}
}

func (s *initResourcePoolServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if Pool(ctx) == nil {
		ctx = WithPool(ctx, struct {
			*resource.Pool
			*sync.Mutex
		}{
			s.resourcePool,
			&s.lock,
		})
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *initResourcePoolServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
