// Copyright (c) 2021 Nordix Foundation.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// +build linux

package token

import (
	"context"
	"os"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/tools/tokens"
)

type tokenServer struct {
	tokenName   string
	sharedToken string
	config      tokenConfig
}

// NewServer returns a new token server chain element for the given tokenKey
func NewServer(tokenKey string) networkservice.NetworkServiceServer {
	sriovTokens := tokens.FromEnv(os.Environ())[tokenKey]
	if len(sriovTokens) == 1 {
		return &tokenServer{
			sharedToken: sriovTokens[0],
		}
	}
	return &tokenServer{
		tokenName: tokenKey,
		config: createTokenElement(map[string][]string{
			tokenKey: sriovTokens,
		}),
	}
}

func (s *tokenServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	isEstablished := s.config.get(request.GetConnection()) != ""

	var tokenID string
	var isEstablished bool
	mechanism := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if mechanism != nil && mechanism.GetDeviceTokenID() == "" {
		if s.sharedToken != "" {
			mechanism.SetDeviceTokenID(s.sharedToken)
		} else if tokenID = s.config.assign(s.tokenName, request.GetConnection()); tokenID != "" {
			mechanism.SetDeviceTokenID(tokenID)
		}
	} else if mechanism != nil && mechanism.GetDeviceTokenID() != "" {
		isEstablished = true
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && tokenID != "" && !isEstablished {
		s.config.release(request.GetConnection())
	}

	return conn, err
}

func (s *tokenServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		s.config.release(conn)
	}
	return next.Server(ctx).Close(ctx, conn)
}
