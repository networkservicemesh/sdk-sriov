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

package token_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/tokens"
)

const (
	tokenName = "service.domain/10G"
	tokenID1  = "sriov-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx1"
	tokenID2  = "sriov-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx2"
)

func TestSharedTokenServer_Request(t *testing.T) {
	name, value := tokens.ToEnv(tokenName, []string{tokenID1})
	err := os.Setenv(name, value)
	require.NoError(t, err)

	server := chain.NewNetworkServiceServer(
		token.NewServer(tokenName),
	)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id1",
			Mechanism: &networkservice.Mechanism{
				Type:       kernel.MECHANISM,
				Parameters: map[string]string{},
			},
		},
	})
	require.NoError(t, err)

	conn2, err2 := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id2",
			Mechanism: &networkservice.Mechanism{
				Type:       kernel.MECHANISM,
				Parameters: map[string]string{},
			},
		},
	})
	require.NoError(t, err2)

	mech := kernel.ToMechanism(conn.GetMechanism())
	require.NotNil(t, mech)
	require.Equal(t, tokenID1, mech.GetDeviceTokenID())

	mech2 := kernel.ToMechanism(conn2.GetMechanism())
	require.NotNil(t, mech2)
	require.Equal(t, tokenID1, mech2.GetDeviceTokenID())
}

func TestMultiTokenServer_Request(t *testing.T) {
	tokenList := []string{tokenID1, tokenID2}
	name, value := tokens.ToEnv(tokenName, tokenList)
	err := os.Setenv(name, value)
	require.NoError(t, err)

	server := chain.NewNetworkServiceServer(
		token.NewServer(tokenName),
	)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id1",
			Mechanism: &networkservice.Mechanism{
				Type:       kernel.MECHANISM,
				Parameters: map[string]string{},
			},
		},
	})
	require.NoError(t, err)

	conn2, err2 := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id2",
			Mechanism: &networkservice.Mechanism{
				Type:       kernel.MECHANISM,
				Parameters: map[string]string{},
			},
		},
	})
	require.NoError(t, err2)

	mech := kernel.ToMechanism(conn.GetMechanism())
	require.NotNil(t, mech)
	require.Subset(t, tokenList, []string{mech.GetDeviceTokenID()})

	mech2 := kernel.ToMechanism(conn2.GetMechanism())
	require.NotNil(t, mech2)
	require.Subset(t, tokenList, []string{mech2.GetDeviceTokenID()})

	require.NotEqual(t, mech.GetDeviceTokenID(), mech2.GetDeviceTokenID())

	conn3, err3 := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id3",
			Mechanism: &networkservice.Mechanism{
				Type:       kernel.MECHANISM,
				Parameters: map[string]string{},
			},
		},
	})
	require.NoError(t, err3)
	mech3 := kernel.ToMechanism(conn3.GetMechanism())
	require.NotNil(t, mech3)
	require.Equal(t, "", mech3.GetDeviceTokenID())
}
