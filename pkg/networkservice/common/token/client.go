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

// Package token provides chain element for inserting SRIOV tokens into request
package token

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/tokens"
)

const (
	sriovTokenLabel    = "sriovToken"
	serviceDomainLabel = "serviceDomain"
)

type tokenClient struct {
	lock                sync.Mutex
	tokens              map[string][]string // tokens[tokenName] -> []tokenIDs
	connectionsByTokens map[string]string   // connectionsByTokens[tokenID] -> connectionID
	tokensByConnections map[string]string   // tokensByConnections[connectionID] -> tokenID
}

// NewClient returns a new token client chain element
func NewClient() networkservice.NetworkServiceClient {
	return &tokenClient{
		tokens:              tokens.FromEnv(os.Environ()),
		connectionsByTokens: map[string]string{},
		tokensByConnections: map[string]string{},
	}
}

func (c *tokenClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if labels := request.GetConnection().GetLabels(); labels != nil {
		if tokenName, ok := labels[sriovTokenLabel]; ok {
			var tokenID string
			for _, tokenID = range c.tokens[tokenName] {
				if _, ok := c.connectionsByTokens[tokenID]; !ok {
					c.connectionsByTokens[tokenID] = request.GetConnection().GetId()
					c.tokensByConnections[request.GetConnection().GetId()] = tokenID
					break
				} else {
					tokenID = ""
				}
			}
			if tokenID == "" {
				return nil, errors.Errorf("no free token for the name: %v", tokenName)
			}

			request = request.Clone()
			delete(request.GetConnection().GetLabels(), sriovTokenLabel)
			request.GetConnection().GetLabels()[serviceDomainLabel] = strings.Split(tokenName, "/")[0]

			for _, mech := range request.GetMechanismPreferences() {
				if mech.Parameters == nil {
					mech.Parameters = map[string]string{}
				}
				mech.Parameters[resourcepool.TokenIDKey] = tokenID
			}
		}
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *tokenClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if tokenID, ok := c.tokensByConnections[conn.GetId()]; ok {
		delete(c.connectionsByTokens, tokenID)
		delete(c.tokensByConnections, conn.GetId())
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
