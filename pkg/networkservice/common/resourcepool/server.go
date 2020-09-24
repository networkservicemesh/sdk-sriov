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
	// CapabilityLabel is a label for capability
	CapabilityLabel = "capability"
)

// PCIResourcePool is a resourcepool.ResourcePool + sync.Locker interface
type PCIResourcePool interface {
	Select(driverType sriov.DriverType, service string, capability sriov.Capability) (string, error)
	Free(vfPciAddr string) error

	sync.Locker
}

type resourcePoolServer struct {
	driverType  sriov.DriverType
	functions   map[sriov.PCIFunction][]sriov.PCIFunction
	binders     map[uint][]sriov.DriverBinder
	selectedVFs map[string]string
}

// NewServer returns a new resource pool server chain element
func NewServer(driverType sriov.DriverType, functions map[sriov.PCIFunction][]sriov.PCIFunction, binders map[uint][]sriov.DriverBinder) networkservice.NetworkServiceServer {
	return &resourcePoolServer{
		driverType:  driverType,
		functions:   functions,
		binders:     binders,
		selectedVFs: map[string]string{},
	}
}

func (s *resourcePoolServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logEntry := log.Entry(ctx).WithField("resourcePoolServer", "Request")

	resourcePool := ResourcePool(ctx)
	if resourcePool == nil {
		return nil, errors.New("ResourcePool not found")
	}

	service, capability, err := getServiceAndCapability(request)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid service: %v", request.GetConnection().GetNetworkService())
	}

	vfConfig := vfconfig.Config(ctx)
	if err := func() error {
		resourcePool.Lock()
		defer resourcePool.Unlock()

		logEntry.Infof("trying to select VF for %v://%v:%v", s.driverType, service, capability)
		vf, err := s.selectVf(request.GetConnection().GetId(), vfConfig, resourcePool, service, capability)
		if err != nil {
			return err
		}
		logEntry.Infof("selected VF: %+v", vf)

		igid, err := vf.GetIommuGroupID()
		if err != nil {
			return errors.Wrapf(err, "failed to get VF IOMMU group: %v", vf.GetPCIAddress())
		}

		if err := s.bindDriver(igid); err != nil {
			return err
		}

		if s.driverType == sriov.VfioPCIDriver {
			vfio.ToMechanism(request.GetConnection().GetMechanism()).SetIommuGroup(igid)
		}

		return nil
	}(); err != nil {
		_ = s.close(ctx, request.GetConnection())
		return nil, err
	}

	return next.Server(ctx).Request(ctx, request)
}

func getServiceAndCapability(request *networkservice.NetworkServiceRequest) (string, sriov.Capability, error) {
	service := request.GetConnection().GetNetworkService()

	var capability sriov.Capability
	if labels := request.GetConnection().GetLabels(); labels == nil {
		capability = sriov.ZeroCapability
	} else if capabilityString, ok := labels[CapabilityLabel]; !ok {
		capability = sriov.ZeroCapability
	} else {
		capability = sriov.Capability(capabilityString)
	}
	if err := capability.Validate(); err != nil {
		return "", "", err
	}

	return service, capability, nil
}

func (s *resourcePoolServer) selectVf(
	connID string,
	vfConfig *vfconfig.VFConfig,
	resourcePool PCIResourcePool,
	service string,
	capability sriov.Capability,
) (vf sriov.PCIFunction, err error) {
	s.selectedVFs[connID], err = resourcePool.Select(s.driverType, service, capability)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to select VF for: %v, %v, %v", s.driverType, service, capability)
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
		case sriov.VfioPCIDriver:
			err = binder.BindDriver(string(sriov.VfioPCIDriver))
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
	vfPciAddr, ok := s.selectedVFs[conn.GetId()]
	if !ok {
		return nil
	}
	delete(s.selectedVFs, conn.GetId())

	resourcePool := ResourcePool(ctx)
	resourcePool.Lock()
	defer resourcePool.Unlock()

	return resourcePool.Free(vfPciAddr)
}
