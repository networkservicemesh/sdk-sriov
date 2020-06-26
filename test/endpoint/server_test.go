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

package endpoint

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newClient(ctx context.Context, u *url.URL) (*grpc.ClientConn, error) {
	clientCtx, clientCancelFunc := context.WithTimeout(ctx, 10*time.Second)
	defer clientCancelFunc()
	return grpc.DialContext(clientCtx, grpcutils.URLToTarget(u),
		grpc.WithInsecure(),
		grpc.WithBlock())
}

func TokenGenerator(_ credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}
func TestEndpoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	testURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	testRequest := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:00:00:0"}},
			{Cls: cls.LOCAL, Type: kernel.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:03:00:0"}},
			{Cls: cls.LOCAL, Type: vfio.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:04:00:0"}},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service",
			Context:        &networkservice.ConnectionContext{},
			Mechanism:      &networkservice.Mechanism{},
		},
	}

	testRequestBad := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:00:00:0"}},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service",
			Context:        &networkservice.ConnectionContext{},
			Mechanism:      &networkservice.Mechanism{},
		},
	}

	// start server
	server, errCh := NewServer(ctx, testURL)
	require.NotNil(t, server)
	require.NotNil(t, errCh)

	// client send request
	var nsmClient grpc.ClientConnInterface
	nsmClient, err := newClient(ctx, testURL)
	require.Nil(t, err)
	cl := client.NewClient(context.Background(), "nsc-1", nil, TokenGenerator, nsmClient)

	var connection *networkservice.Connection

	connection, err = cl.Request(ctx, testRequest)
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, "0000:03:00:0", connection.Mechanism.Parameters[kernel.PCIAddress])

	connection, err = cl.Request(ctx, testRequest)
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, "0000:04:00:0", connection.Mechanism.Parameters[kernel.PCIAddress])

	_, err = cl.Request(ctx, testRequestBad)
	require.NotNil(t, err)
}
