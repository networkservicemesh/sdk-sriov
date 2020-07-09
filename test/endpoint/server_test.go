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

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/common/selectorpciaddress"

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
		Context: &networkservice.ConnectionContext{
			ExtraContext: map[string]string{
				selectorpciaddress.FreeVirtualFunctionsInfoKey: "FreeVirtualFunctions:\n  \"0000:01:00:0\": 2\n  \"0000:02:00:0\": 1\n  \"0000:03:00:0\": 0\n",
			},
		},
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
		{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		{Cls: cls.LOCAL, Type: vfio.MECHANISM},
	}

	testConn0 := getConnectionSetting("0")

	testURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	testRequest := &networkservice.NetworkServiceRequest{
		MechanismPreferences: mechPref,
		Connection:           testConn0,
	}

	config, err := sriov.ReadConfig(context.Background(), configFileName)
	require.Nil(t, err)
	// start server
	endpoint := NewServer("server", authorize.NewServer(), tokenGenerator, testURL, config)

	// test request
	connection, err := endpoint.Request(context.Background(), testRequest)
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, "0000:01:00:0", connection.Mechanism.Parameters[kernel.PCIAddress])
}
