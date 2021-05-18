// Copyright (c) 2021 Nordix Foundation.
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

package resourcepool

import (
	"context"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

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

type resourcePoolConfig struct {
	driverType   sriov.DriverType
	resourceLock sync.Locker
	pciPool      PCIPool
	resourcePool ResourcePool
	config       *config.Config
	selectedVFs  map[string]string
}

func (s *resourcePoolConfig) selectVF(connID string, vfConfig *vfconfig.VFConfig, tokenID string) (vf sriov.PCIFunction, err error) {
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

func (s *resourcePoolConfig) close(conn *networkservice.Connection) error {
	vfPCIAddr, ok := s.selectedVFs[conn.GetId()]
	if !ok {
		return nil
	}
	delete(s.selectedVFs, conn.GetId())

	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	return s.resourcePool.Free(vfPCIAddr)
}

func assignVF(ctx context.Context, logger log.Logger, conn *networkservice.Connection, tokenID string, resourcePool *resourcePoolConfig) error {
	vfConfig := vfconfig.Config(ctx)
	resourcePool.resourceLock.Lock()
	defer resourcePool.resourceLock.Unlock()

	logger.Infof("trying to select VF for %v", resourcePool.driverType)
	vf, err := resourcePool.selectVF(conn.GetId(), vfConfig, tokenID)
	if err != nil {
		return err
	}
	logger.Infof("selected VF: %+v", vf)

	iommuGroup, err := vf.GetIOMMUGroup()
	if err != nil {
		return errors.Wrapf(err, "failed to get VF IOMMU group: %v", vf.GetPCIAddress())
	}

	if err = resourcePool.pciPool.BindDriver(ctx, iommuGroup, resourcePool.driverType); err != nil {
		return err
	}

	switch resourcePool.driverType {
	case sriov.KernelDriver:
		vfConfig.VFInterfaceName, err = vf.GetNetInterfaceName()
		if err != nil {
			return errors.Wrapf(err, "failed to get VF net interface name: %v", vf.GetPCIAddress())
		}
	case sriov.VFIOPCIDriver:
		vfio.ToMechanism(conn.GetMechanism()).SetIommuGroup(iommuGroup)
	}

	return nil
}
