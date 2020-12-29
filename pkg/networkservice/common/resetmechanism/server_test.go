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

package resetmechanism_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resetmechanism"
)

const (
	mech1 = "mech-1"
	mech2 = "mech-2"
)

func testRequest(mechType string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "test-ID",
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: mechType,
			},
		},
	}
}

func TestResetMechanismServer_Request_Update(t *testing.T) {
	mechElement := newMechChainElement()
	mockElement := newMockChainElement()

	server := chain.NewNetworkServiceServer(
		resetmechanism.NewServer(mechElement),
		mockElement,
	)

	// 1. Request with mech1 mechanism

	request := testRequest(mech1)

	conn, err := server.Request(context.TODO(), request)
	require.NoError(t, err)
	require.Equal(t, mech1, conn.Mechanism.Type)

	require.True(t, mechElement.mechs[mech1])

	mechElement.mock.AssertNumberOfCalls(t, "Request", 1)
	mechElement.mock.AssertNumberOfCalls(t, "Close", 0)
	mockElement.mock.AssertNumberOfCalls(t, "Request", 1)
	mockElement.mock.AssertNumberOfCalls(t, "Close", 0)

	// 2. Update request

	conn, err = server.Request(context.TODO(), request)
	require.NoError(t, err)

	mechElement.mock.AssertNumberOfCalls(t, "Request", 1)
	mechElement.mock.AssertNumberOfCalls(t, "Close", 0)
	mockElement.mock.AssertNumberOfCalls(t, "Request", 2)
	mockElement.mock.AssertNumberOfCalls(t, "Close", 0)

	// 3. Close

	_, err = server.Close(context.TODO(), conn)
	require.NoError(t, err)

	require.False(t, mechElement.mechs[mech1])
	require.False(t, mechElement.mechs[mech2])

	mechElement.mock.AssertNumberOfCalls(t, "Request", 1)
	mechElement.mock.AssertNumberOfCalls(t, "Close", 1)
	mockElement.mock.AssertNumberOfCalls(t, "Request", 2)
	mockElement.mock.AssertNumberOfCalls(t, "Close", 1)
}

func TestResetMechanismServer_Request_Change(t *testing.T) {
	mechElement := newMechChainElement()
	mockElement := newMockChainElement()

	server := chain.NewNetworkServiceServer(
		resetmechanism.NewServer(mechElement),
		mockElement,
	)

	// 1. Request with mech1 mechanism

	conn, err := server.Request(context.TODO(), testRequest(mech1))
	require.NoError(t, err)
	require.Equal(t, mech1, conn.Mechanism.Type)

	require.True(t, mechElement.mechs[mech1])

	mechElement.mock.AssertNumberOfCalls(t, "Request", 1)
	mechElement.mock.AssertNumberOfCalls(t, "Close", 0)
	mockElement.mock.AssertNumberOfCalls(t, "Request", 1)
	mockElement.mock.AssertNumberOfCalls(t, "Close", 0)

	// 2. Request with mech2 mechanism

	conn, err = server.Request(context.TODO(), testRequest(mech2))
	require.NoError(t, err)
	require.Equal(t, mech2, conn.Mechanism.Type)

	require.False(t, mechElement.mechs[mech1])
	require.True(t, mechElement.mechs[mech2])

	mechElement.mock.AssertNumberOfCalls(t, "Request", 2)
	mechElement.mock.AssertNumberOfCalls(t, "Close", 1)
	mockElement.mock.AssertNumberOfCalls(t, "Request", 2)
	mockElement.mock.AssertNumberOfCalls(t, "Close", 0)

	// 3. Close

	_, err = server.Close(context.TODO(), conn)
	require.NoError(t, err)

	require.False(t, mechElement.mechs[mech1])
	require.False(t, mechElement.mechs[mech2])

	mechElement.mock.AssertNumberOfCalls(t, "Request", 2)
	mechElement.mock.AssertNumberOfCalls(t, "Close", 2)
	mockElement.mock.AssertNumberOfCalls(t, "Request", 2)
	mockElement.mock.AssertNumberOfCalls(t, "Close", 1)
}

type mechChainElement struct {
	mock  mock.Mock
	mechs map[string]bool
}

func newMechChainElement() *mechChainElement {
	m := &mechChainElement{
		mechs: map[string]bool{},
	}

	m.mock.On("Request", mock.Anything, mock.Anything).
		Return(nil, nil)
	m.mock.On("Close", mock.Anything, mock.Anything).
		Return(nil, nil)

	return m
}

func (m *mechChainElement) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	m.mock.Called(ctx, request)

	mech := request.GetMechanismPreferences()[0]
	request.GetConnection().Mechanism = mech
	m.mechs[mech.GetType()] = true

	return next.Server(ctx).Request(ctx, request)
}

func (m *mechChainElement) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	m.mock.Called(ctx, conn)

	mech := conn.GetMechanism()
	m.mechs[mech.GetType()] = false

	return next.Server(ctx).Close(ctx, conn)
}

type mockChainElement struct {
	mock mock.Mock
}

func (m *mockChainElement) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	m.mock.Called(ctx, request)
	return next.Server(ctx).Request(ctx, request)
}

func (m *mockChainElement) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	m.mock.Called(ctx, conn)
	return next.Server(ctx).Close(ctx, conn)
}

func newMockChainElement() *mockChainElement {
	m := &mockChainElement{}

	m.mock.On("Request", mock.Anything, mock.Anything).
		Return(nil, nil)
	m.mock.On("Close", mock.Anything, mock.Anything).
		Return(nil, nil)

	return m
}
