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

// Package selectormechanism provides a selection from mechanisms list
package selectormechanism

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type selectMechServer struct {
	selector *roundRobinSelector
}

// NewServer - select mechanism
func NewServer() networkservice.NetworkServiceServer {
	return &selectMechServer{
		selector: newRoundRobinSelector(),
	}
}

func (s *selectMechServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mechanisms := request.GetMechanismPreferences()
	if len(mechanisms) > 0 {
		mech := s.selector.selectMechanism(request.GetMechanismPreferences())

		if request.GetConnection() == nil {
			request.Connection = &networkservice.Connection{Id: "Id"}
		}

		if request.GetConnection().GetMechanism() == nil {
			request.GetConnection().Mechanism = &networkservice.Mechanism{}
		}
		request.GetConnection().Mechanism = mech
	} else {
		return nil, errors.New("mechanisms are empty")
	}

	return next.Server(ctx).Request(ctx, request)
}

func (s *selectMechServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
