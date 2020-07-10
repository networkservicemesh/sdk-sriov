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

// Package vfio provides a networkservice chain element that properly handles the SR-IOV VFIO Mechanism
package vfio

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

type vfioServer struct {
	resourcePool *sriov.NetResourcePool
}

// NewServer return a NetworkServiceServer chain element that correctly handles the VFIO Mechanism
func NewServer(resourcePool *sriov.NetResourcePool) networkservice.NetworkServiceServer {
	return &vfioServer{
		resourcePool: resourcePool,
	}
}

func (a *vfioServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	pciAddress, ok := conn.GetMechanism().GetParameters()[vfio.PCIAddress]
	if !ok {
		return nil, errors.Errorf("No selected physical function provided")
	}

	_, err = a.resourcePool.SelectVirtualFunction(pciAddress)
	if err != nil {
		return nil, err
	}

	// TODO fill required VFIO fields in Connection

	return conn, nil
}

func (a *vfioServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, ok := conn.GetMechanism().GetParameters()[vfio.PCIAddress]
	if !ok {
		return nil, errors.Errorf("No physical function PCI address found")
	}

	// TODO get required VFIO fields from Connection and release virtual function

	return next.Server(ctx).Close(ctx, conn)
}
