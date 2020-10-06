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

// Package resourcepool provides chain elements for to select and free VF
package resourcepool

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

const (
	// TokenIDKey is a token ID mechanism parameter key
	TokenIDKey = "tokenID" // TODO: move to api
)

// ResourcePool is a resource.Pool interface
type ResourcePool interface {
	Select(tokenID string, driverType sriov.DriverType) (string, error)
	Free(vfPCIAddr string) error
}

type resourcePoolServer struct {
	driverType   sriov.DriverType
	resourceLock sync.Locker
	functions    map[sriov.PCIFunction][]sriov.PCIFunction
	binders      map[uint][]sriov.DriverBinder
	selectedVFs  map[string]string
}

// NewServer returns a new resource pool server chain element
func NewServer(
	driverType sriov.DriverType,
	resourceLock sync.Locker,
	functions map[sriov.PCIFunction][]sriov.PCIFunction,
	binders map[uint][]sriov.DriverBinder,
) networkservice.NetworkServiceServer {
	return &resourcePoolServer{
		driverType:   driverType,
		resourceLock: resourceLock,
		functions:    functions,
		binders:      binders,
		selectedVFs:  map[string]string{},
	}
}

func (s *resourcePoolServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logEntry := log.Entry(ctx).WithField("resourcePoolServer", "Request")

	resourcePool := Pool(ctx)
	if resourcePool == nil {
		return nil, errors.New("ResourcePool not found")
	}

	tokenID, ok := request.GetConnection().GetMechanism().GetParameters()[TokenIDKey]
	if !ok {
		return nil, errors.New("no token ID provided")
	}

	vfConfig := vfconfig.Config(ctx)
	if err := func() error {
		s.resourceLock.Lock()
		defer s.resourceLock.Unlock()

		logEntry.Infof("trying to select VF for %v", s.driverType)
		vf, err := s.selectVF(request.GetConnection().GetId(), vfConfig, resourcePool, tokenID)
		if err != nil {
			return err
		}
		logEntry.Infof("selected VF: %+v", vf)

		iommuGroup, err := vf.GetIOMMUGroup()
		if err != nil {
			return errors.Wrapf(err, "failed to get VF IOMMU group: %v", vf.GetPCIAddress())
		}

		if err := s.bindDriver(iommuGroup); err != nil {
			return err
		}

		if s.driverType == sriov.VFIOPCIDriver {
			vfio.ToMechanism(request.GetConnection().GetMechanism()).SetIommuGroup(iommuGroup)
		}

		return nil
	}(); err != nil {
		_ = s.close(ctx, request.GetConnection())
		return nil, err
	}

	return next.Server(ctx).Request(ctx, request)
}

func (s *resourcePoolServer) selectVF(
	connID string,
	vfConfig *vfconfig.VFConfig,
	resourcePool ResourcePool,
	tokenID string,
) (vf sriov.PCIFunction, err error) {
	s.selectedVFs[connID], err = resourcePool.Select(tokenID, s.driverType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to select VF for: %v", s.driverType)
	}

	for pf, vfs := range s.functions {
		for i, vf := range vfs {
			if vf.GetPCIAddress() != s.selectedVFs[connID] {
				continue
			}

			vfConfig.VFInterfaceName, err = vf.GetNetInterfaceName()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get VF net interface name: %v", vf.GetPCIAddress())
			}
			vfConfig.PFInterfaceName, err = pf.GetNetInterfaceName()
			if err != nil {
				return nil, errors.Errorf("failed to get PF net interface name: %v", pf.GetPCIAddress())
			}
			vfConfig.VFNum = i

			return vf, nil
		}
	}

	return nil, errors.Errorf("no VF with selected PCI address exists: %v", s.selectedVFs[connID])
}

func (s *resourcePoolServer) bindDriver(igid uint) (err error) {
	for _, binder := range s.binders[igid] {
		switch s.driverType {
		case sriov.KernelDriver:
			err = binder.BindKernelDriver()
		case sriov.VFIOPCIDriver:
			err = binder.BindDriver(string(sriov.VFIOPCIDriver))
		}
		if err != nil {
			return errors.Wrapf(err, "failed to bind driver to IOMMU group: %v", igid)
		}
	}
	return nil
}

func (s *resourcePoolServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	closeErr := s.close(ctx, conn)

	rv, err := next.Server(ctx).Close(ctx, conn)
	if err != nil && closeErr != nil {
		return rv, errors.Wrapf(err, "failed to free VF: %v", closeErr)
	}
	if closeErr != nil {
		return rv, errors.Wrap(closeErr, "failed to free VF")
	}
	return rv, err
}

func (s *resourcePoolServer) close(ctx context.Context, conn *networkservice.Connection) error {
	vfPCIAddr, ok := s.selectedVFs[conn.GetId()]
	if !ok {
		return nil
	}
	delete(s.selectedVFs, conn.GetId())

	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	return Pool(ctx).Free(vfPCIAddr)
}
