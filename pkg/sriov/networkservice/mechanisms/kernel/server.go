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
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/utils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	sdkKernel "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
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

func (a *kernelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	pciAddress, ok := conn.GetMechanism().GetParameters()[kernel.PCIAddress]
	if !ok {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, errors.Errorf("No selected physical function provided")
	}

	selectedVf, err := a.resourcePool.SelectVirtualFunction(pciAddress)
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

	clientNetNSHandle, err := getClientNetNSHandle(request.GetConnection())
	if err != nil {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}

	err = moveInterfaceToAnotherNamespace(ifaceName, forwarderNetNSHandle, clientNetNSHandle)
	if err != nil {
		_, _ = next.Server(ctx).Close(ctx, conn)
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Client's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Client's namespace for connection %s", ifaceName, request.GetConnection().GetId())

	return conn, nil
}

func (a *kernelServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, _ = next.Server(ctx).Close(ctx, conn)

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

	clientNetNSHandle, err := getClientNetNSHandle(conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}

	err = moveInterfaceToAnotherNamespace(ifaceName, clientNetNSHandle, forwarderNetNSHandle)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Forwarder's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Forwarder's namespace for connection %s", ifaceName, conn.GetId())

	err = a.resourcePool.ReleaseVirtualFunction(pciAddress, ifaceName)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func moveInterfaceToAnotherNamespace(ifaceName string, fromNetNS, toNetNS netns.NsHandle) error {
	link, err := sdkKernel.FindHostDevice("", ifaceName, fromNetNS)
	if err != nil {
		return err
	}

	err = link.MoveToNetns(toNetNS)
	if err != nil {
		return errors.Wrapf(err, "Failed to move interface %s to another namespace", ifaceName)
	}

	return nil
}

func getClientNetNSHandle(conn *networkservice.Connection) (netns.NsHandle, error) {
	clientNetNSInode := conn.GetMechanism().GetParameters()[kernel.NetNSInodeKey]
	if clientNetNSInode == "" {
		return 0, errors.New("Client's pod net ns inode is not found")
	}

	return utils.GetNSHandleFromInode(clientNetNSInode)
}
