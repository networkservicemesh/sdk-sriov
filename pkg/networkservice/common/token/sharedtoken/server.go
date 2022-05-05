// Copyright (c) 2021-2022 Nordix Foundation.
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

//go:build linux
// +build linux

// Package sharedtoken provides server chain element for inserting shared SRIOV token into request and response
package sharedtoken

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type tokenServer struct {
	sharedToken string
}

// NewServer returns a new shard token server chain element for the given token
func NewServer(token string) networkservice.NetworkServiceServer {
	return &tokenServer{
		sharedToken: token,
	}
}

func (s *tokenServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mechanism := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if mechanism != nil && mechanism.GetDeviceTokenID() == "" {
		mechanism.SetDeviceTokenID(s.sharedToken)
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *tokenServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
