// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
)

const (
	// TokenIDKey is a token ID mechanism parameter key
	TokenIDKey = "tokenID" // TODO: move to api
)

// PCIPool is a pci.Pool interface
type PCIPool interface {
	GetPCIFunction(pciAddr string) (sriov.PCIFunction, error)
	BindDriver(ctx context.Context, iommuGroup uint, driverType sriov.DriverType) error
}

// ResourcePool is a resource.Pool interface
type ResourcePool interface {
	Select(tokenID string, driverType sriov.DriverType) (string, error)
	Free(vfPCIAddr string) error
}

type resourcePoolServer struct {
	driverType   sriov.DriverType
	resourceLock sync.Locker
	pciPool      PCIPool
	resourcePool ResourcePool
	config       *config.Config
	selectedVFs  map[string]string
}

// NewServer returns a new resource pool server chain element
func NewServer(
	driverType sriov.DriverType,
	resourceLock sync.Locker,
	pciPool PCIPool,
	resourcePool ResourcePool,
	cfg *config.Config,
) networkservice.NetworkServiceServer {
	return &resourcePoolServer{
		driverType:   driverType,
		resourceLock: resourceLock,
		pciPool:      pciPool,
		resourcePool: resourcePool,
		config:       cfg,
		selectedVFs:  map[string]string{},
	}
}

func (s *resourcePoolServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("resourcePoolServer", "Request")

	tokenID, ok := request.GetConnection().GetMechanism().GetParameters()[TokenIDKey]
	if !ok {
		return nil, errors.New("no token ID provided")
	}

	vfConfig := vfconfig.Config(ctx)
	if err := func() error {
		s.resourceLock.Lock()
		defer s.resourceLock.Unlock()

		logger.Infof("trying to select VF for %v", s.driverType)
		vf, err := s.selectVF(request.GetConnection().GetId(), vfConfig, tokenID)
		if err != nil {
			return err
		}
		logger.Infof("selected VF: %+v", vf)

		iommuGroup, err := vf.GetIOMMUGroup()
		if err != nil {
			return errors.Wrapf(err, "failed to get VF IOMMU group: %v", vf.GetPCIAddress())
		}

		if err = s.pciPool.BindDriver(ctx, iommuGroup, s.driverType); err != nil {
			return err
		}

		switch s.driverType {
		case sriov.KernelDriver:
			vfConfig.VFInterfaceName, err = vf.GetNetInterfaceName()
			if err != nil {
				return errors.Wrapf(err, "failed to get VF net interface name: %v", vf.GetPCIAddress())
			}
		case sriov.VFIOPCIDriver:
			vfio.ToMechanism(request.GetConnection().GetMechanism()).SetIommuGroup(iommuGroup)
		}

		return nil
	}(); err != nil {
		_ = s.close(request.GetConnection())
		return nil, err
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		_ = s.close(request.GetConnection())
	}

	return conn, err
}

func (s *resourcePoolServer) selectVF(connID string, vfConfig *vfconfig.VFConfig, tokenID string) (vf sriov.PCIFunction, err error) {
	vfPCIAddr, err := s.resourcePool.Select(tokenID, s.driverType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to select VF for: %v", s.driverType)
	}
	s.selectedVFs[connID] = vfPCIAddr

	for pfPCIAddr, pfCfg := range s.config.PhysicalFunctions {
		for i, vfCfg := range pfCfg.VirtualFunctions {
			if vfCfg.Address != vfPCIAddr {
				continue
			}

			pf, err := s.pciPool.GetPCIFunction(pfPCIAddr)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get PF: %v", pfPCIAddr)
			}
			vfConfig.PFInterfaceName, err = pf.GetNetInterfaceName()
			if err != nil {
				return nil, errors.Errorf("failed to get PF net interface name: %v", pfPCIAddr)
			}

			vf, err := s.pciPool.GetPCIFunction(vfPCIAddr)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get VF: %v", vfPCIAddr)
			}

			vfConfig.VFNum = i

			return vf, err
		}
	}

	return nil, errors.Errorf("no VF with selected PCI address exists: %v", s.selectedVFs[connID])
}

func (s *resourcePoolServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)

	closeErr := s.close(conn)

	if err != nil && closeErr != nil {
		return nil, errors.Wrapf(err, "failed to free VF: %v", closeErr)
	}
	if closeErr != nil {
		return nil, errors.Wrap(closeErr, "failed to free VF")
	}
	return &empty.Empty{}, err
}

func (s *resourcePoolServer) close(conn *networkservice.Connection) error {
	vfPCIAddr, ok := s.selectedVFs[conn.GetId()]
	if !ok {
		return nil
	}
	delete(s.selectedVFs, conn.GetId())

	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	return s.resourcePool.Free(vfPCIAddr)
}
