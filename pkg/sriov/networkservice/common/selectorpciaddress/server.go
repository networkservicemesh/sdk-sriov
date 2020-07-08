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

// Package selectorpciaddress provides a selection from mechanisms list
package selectorpciaddress

import (
	"context"
	"errors"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/networkservice/common/filterpciaddress"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type selectPCIAddrServer struct {
	selector         *roundRobinSelector
	connectedPCIAddr sync.Map
}

// NewServer - select mechanism
func NewServer() networkservice.NetworkServiceServer {
	return &selectPCIAddrServer{
		selector: newRoundRobinSelector(),
	}
}

func (s *selectPCIAddrServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	list := filterpciaddress.PCIAddrList(ctx)
	pciAddr := ""
	if len(list) > 0 {
		// selection
		if pciAddr = s.selector.selectStringItem(list); pciAddr != "" {
			if request.GetConnection() == nil {
				return nil, errors.New("request connection is invalid")
			}

			if request.GetConnection().GetMechanism() == nil {
				return nil, errors.New("request connection is invalid")
			}

			if request.GetConnection().GetMechanism().GetParameters() == nil {
				return nil, errors.New("request connection is invalid")
			}

			// set pci address for response
			if mType := filtermechanisms.SelectedMechanismType(ctx); mType != "" {
				switch mType {
				case kernel.MECHANISM:
					request.GetConnection().GetMechanism().GetParameters()[kernel.PCIAddress] = pciAddr
				case vfio.MECHANISM:
					request.GetConnection().GetMechanism().GetParameters()[vfio.PCIAddress] = pciAddr
				}
			} else {
				return nil, errors.New("selected mechanism type is invalid")
			}
		} else {
			return nil, errors.New("pci address selection failed")
		}
	} else {
		return nil, errors.New("pci address list is empty")
	}
	connection, err := next.Server(ctx).Request(ctx, request)
	if err == nil && pciAddr != "" {
		// mark connection Id with corresponding pci address
		s.connectedPCIAddr.Store(connection.GetId(), pciAddr)
		// decrement
		if freeVFInfo := filterpciaddress.FreeVFInfo(ctx); freeVFInfo != nil {
			freeVFInfo.FreeVirtualFunctions[pciAddr]--
		}
	}

	return connection, err
}

func (s *selectPCIAddrServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	emptyValue, err := next.Server(ctx).Close(ctx, conn)
	if err == nil {
		if val, ok := s.connectedPCIAddr.Load(conn.GetId()); ok {
			// increment
			pciAddr := val.(string)
			if freeVFInfo := filterpciaddress.FreeVFInfo(ctx); freeVFInfo != nil {
				freeVFInfo.FreeVirtualFunctions[pciAddr]++
			}

			// delete
			s.connectedPCIAddr.Delete(conn.GetId())
		}
	}
	return emptyValue, err
}
