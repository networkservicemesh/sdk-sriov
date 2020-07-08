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

// Package filtermechanisms provides a filtration of supported mechanisms
package filtermechanisms

import (
	"context"
	"errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type filterMechanismsServer struct {
}

// NewServer - filters out mechanisms by type and provided pci address parametr
func NewServer() networkservice.NetworkServiceServer {
	return &filterMechanismsServer{}
}

func (f *filterMechanismsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection() == nil {
		return nil, errors.New("request connection is invalid")
	}

	if request.GetMechanismPreferences() == nil {
		return nil, errors.New("mechanism preferences are invalid")
	}

	// check and select supported mechanism type
	for _, mechanism := range request.GetMechanismPreferences() {
		mtype := mechanism.GetType()
		switch mtype {
		case kernel.MECHANISM:
		case vfio.MECHANISM:
			ctx = WithSelectedMechanismType(ctx, mtype)
		}
	}

	return next.Server(ctx).Request(ctx, request)
}

func (f *filterMechanismsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
