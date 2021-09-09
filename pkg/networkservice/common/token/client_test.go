// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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

// +build linux

package token_test

import (
	"context"
	"os"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/tokens"
)

const (
	tokenName          = "service.domain/10G"
	tokenID            = "1"
	sriovTokenLabel    = "sriovToken"
	serviceDomainLabel = "serviceDomain"
	serviceDomain      = "service.domain"
)

func TestTokenClient_Request(t *testing.T) {
	name, value := tokens.ToEnv(tokenName, []string{tokenID})
	err := os.Setenv(name, value)
	require.NoError(t, err)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
			Labels: map[string]string{
				sriovTokenLabel: tokenName,
			},
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: "a",
			},
			{
				Type: "b",
			},
		},
	}

	client := chain.NewNetworkServiceClient(
		token.NewClient(),
		&validateClient{t},
	)
	_, err = client.Request(context.TODO(), request)
	require.NoError(t, err)

	require.Equal(t, map[string]string{
		sriovTokenLabel: tokenName,
	}, request.GetConnection().GetLabels())
}

type validateClient struct {
	t *testing.T
}

func (c *validateClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	require.Equal(c.t, map[string]string{
		serviceDomainLabel: serviceDomain,
	}, request.GetConnection().GetLabels())

	for _, mech := range request.GetMechanismPreferences() {
		require.Equal(c.t, tokenID, mech.GetParameters()[resourcepool.TokenIDKey])
	}

	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *validateClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
