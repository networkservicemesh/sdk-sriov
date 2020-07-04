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

// Package markermechanism provides a marking of used mechanism
package markermechanism

import (
	"context"
	"errors"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type markerMechanismServer struct {
	mechUsed *sync.Map
}

// NewServer - marker of selected mechanism for corresponding connection Id
func NewServer(mechUsed *sync.Map) networkservice.NetworkServiceServer {
	return &markerMechanismServer{mechUsed: mechUsed}
}

func (m *markerMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mech := request.GetConnection().Mechanism; mech != nil {
		m.mechUsed.Store(request.GetConnection().GetId(), mech)
	} else {
		return nil, errors.New("mechanism is empty")
	}

	return next.Server(ctx).Request(ctx, request)
}

func (m *markerMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	m.mechUsed.Delete(conn.GetId())
	return next.Server(ctx).Close(ctx, conn)
}
