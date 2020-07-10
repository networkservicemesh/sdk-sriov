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

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/utils"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

type kernelServer struct {
	resourcePool   *sriov.NetResourcePool
	kernelProvider utils.KernelProvider
}

// NewServer return a NetworkServiceServer chain element that correctly handles the kernel Mechanism
func NewServer(resourcePool *sriov.NetResourcePool, kernelProvider utils.KernelProvider) networkservice.NetworkServiceServer {
	return &kernelServer{
		resourcePool:   resourcePool,
		kernelProvider: kernelProvider,
	}
}

func (k *kernelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	pciAddress, ok := conn.GetMechanism().GetParameters()[kernel.PCIAddress]
	if !ok {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, errors.Errorf("No selected physical function provided")
	}

	selectedVf, err := k.resourcePool.SelectVirtualFunction(pciAddress)
	if err != nil {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, err
	}
	ifaceName := selectedVf.NetInterfaceName
	conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey] = ifaceName

	forwarderNetNSHandle, err := netns.Get()
	if err != nil {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, errors.Wrapf(err, "Unable to obtain Forwarder's network namespace handle")
	}

	clientNetNSHandle, err := k.getClientNetNSHandle(conn)
	if err != nil {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}

	err = k.kernelProvider.MoveInterfaceToAnotherNamespace(ifaceName, forwarderNetNSHandle, clientNetNSHandle)
	if err != nil {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Client's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Client's namespace for connection %s", ifaceName, request.GetConnection().GetId())

	return conn, nil
}

func (k *kernelServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, errFromNext := next.Server(ctx).Close(ctx, conn)

	pciAddress, ok := conn.GetMechanism().GetParameters()[kernel.PCIAddress]
	if !ok {
		return nil, errors.Errorf("No physical function PCI address found")
	}

	ifaceName, ok := conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey]
	if !ok {
		return nil, errors.Errorf("No net interface name found")
	}

	forwarderNetNSHandle, err := netns.Get()
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Forwarder's network namespace handle")
	}

	clientNetNSHandle, err := k.getClientNetNSHandle(conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}

	err = k.kernelProvider.MoveInterfaceToAnotherNamespace(ifaceName, clientNetNSHandle, forwarderNetNSHandle)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Forwarder's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Forwarder's namespace for connection %s", ifaceName, conn.GetId())

	err = k.resourcePool.ReleaseVirtualFunction(pciAddress, ifaceName)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, errFromNext
}

func (k *kernelServer) getClientNetNSHandle(conn *networkservice.Connection) (netns.NsHandle, error) {
	clientNetNSInode := conn.GetMechanism().GetParameters()[kernel.NetNSInodeKey]
	if clientNetNSInode == "" {
		return 0, errors.New("Client's pod net ns inode is not found")
	}

	return k.kernelProvider.GetNSHandleFromInode(clientNetNSInode)
}
