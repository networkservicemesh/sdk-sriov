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

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"go.uber.org/goleak"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

const (
	configFileName = "../config/config.yml"
)

func tokenGenerator(_ credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

func getConnectionSetting(id string) *networkservice.Connection {
	conn := &networkservice.Connection{
		Id:             id,
		NetworkService: "my-service",
		Context:        &networkservice.ConnectionContext{},
		Mechanism:      &networkservice.Mechanism{},
		Path: &networkservice.Path{
			Index: 0,
			PathSegments: []*networkservice.PathSegment{
				{
					Token:   "my_token",
					Expires: &timestamp.Timestamp{Seconds: time.Now().Add(time.Minute * 10).Unix()},
				},
			},
		},
	}

	return conn
}
func TestEndpointSimple(t *testing.T) {
	defer goleak.VerifyNone(t)
	mechPref := []*networkservice.Mechanism{
		{Cls: cls.LOCAL, Type: kernel.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:00:00:0"}},
		{Cls: cls.LOCAL, Type: kernel.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:03:00:0"}},
		{Cls: cls.LOCAL, Type: vfio.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:04:00:0"}},
	}

	testConn0 := getConnectionSetting("0")

	testURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	testRequest := &networkservice.NetworkServiceRequest{
		MechanismPreferences: mechPref,
		Connection:           testConn0,
	}

	testRequest2 := &networkservice.NetworkServiceRequest{
		MechanismPreferences: mechPref,
		Connection:           getConnectionSetting("1"),
	}

	testRequestBad := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM, Parameters: map[string]string{kernel.PCIAddress: "0000:00:00:0"}},
		},
		Connection: testConn0,
	}

	config, err := sriov.ReadConfig(context.Background(), configFileName)
	require.Nil(t, err)
	// start server
	endpoint := NewServer("server", authorize.NewServer(), tokenGenerator, testURL, config)

	var connection *networkservice.Connection
	// test if not supported mechanisms
	_, err = endpoint.Request(context.Background(), testRequestBad)
	require.NotNil(t, err)

	// test request
	connection, err = endpoint.Request(context.Background(), testRequest)
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, "0000:03:00:0", connection.Mechanism.Parameters[kernel.PCIAddress])

	connection, err = endpoint.Request(context.Background(), testRequestBad)
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, "0000:03:00:0", connection.Mechanism.Parameters[kernel.PCIAddress])

	// test from the same connection id
	connection, err = endpoint.Request(context.Background(), testRequest)
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, "0000:03:00:0", connection.Mechanism.Parameters[kernel.PCIAddress])

	// test selection via round robin
	connection, err = endpoint.Request(context.Background(), testRequest2)
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, "0000:04:00:0", connection.Mechanism.Parameters[kernel.PCIAddress])

	// test on close
	_, err = endpoint.Close(context.Background(), testRequest.Connection)
	require.Nil(t, err)
	connection, err = endpoint.Request(context.Background(), testRequestBad)
	require.NotNil(t, err)
	require.Nil(t, connection)
}
