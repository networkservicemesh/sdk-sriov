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

// Package kernel provides a networkservice chain element that properly handles the SR-IOV kernel Mechanism
package kernel

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

type kernelServer struct {
	resourcePool *sriov.NetResourcePool
}

// NewServer return a NetworkServiceServer chain element that correctly handles the kernel Mechanism
func NewServer(resourcePool *sriov.NetResourcePool) networkservice.NetworkServiceServer {
	return &kernelServer{
		resourcePool: resourcePool,
	}
}

func (k *kernelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	defer func() {
		if err != nil {
			// don't forget to call Close to release allocated resources on Endpoint side
			_, _ = next.Server(ctx).Close(ctx, conn)
		}
	}()

	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	pciAddress, ok := conn.GetMechanism().GetParameters()[kernel.PCIAddress]
	if !ok {
		return nil, errors.Errorf("No selected physical function provided")
	}

	selectedVf, err := k.resourcePool.SelectVirtualFunction(pciAddress)
	if err != nil {
		return nil, err
	}
	conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey] = selectedVf.NetInterfaceName

	return conn, nil
}

func (k *kernelServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	_, err2 := next.Server(ctx).Close(ctx, conn)
	defer func() {
		if err2 != nil {
			// we want to return initial error, not the latest one
			err = err2
		}
	}()

	pciAddress, ok := conn.GetMechanism().GetParameters()[kernel.PCIAddress]
	if !ok {
		return nil, errors.Errorf("No physical function PCI address found")
	}

	ifaceName, ok := conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey]
	if !ok {
		return nil, errors.Errorf("No net interface name found")
	}

	err = k.resourcePool.ReleaseVirtualFunction(pciAddress, ifaceName)
	if err != nil {
		return nil, err
	}

	return
}
