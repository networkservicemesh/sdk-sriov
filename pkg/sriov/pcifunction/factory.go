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

// Package pcifunction provides Linux api/pci_function implementation
package pcifunction

import (
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"

	api "github.com/networkservicemesh/sdk-sriov/pkg/sriov/api/pcifunction"
)

const (
	bdfDomain = "0000:"
)

var validLongPCIAddr = regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]{1}$`)
var validShortPCIAddr = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]{1}$`)

// Factory is a PhysicalFunction factory class
type Factory struct {
	paths paths
}

// NewFactory returns a new Factory
func NewFactory(pciDevicesPath, pciDriversPath, iommuGroupsPath string) api.Factory {
	return &Factory{
		paths: paths{
			pciDevices:  pciDevicesPath,
			pciDrivers:  pciDriversPath,
			iommuGroups: iommuGroupsPath,
		},
	}
}

// NewConfigurablePhysicalFunction returns a new api.ConfigurablePhysicalFunction
func (f *Factory) NewConfigurablePhysicalFunction(pciAddress string) (api.ConfigurablePhysicalFunction, error) {
	return f.newPhysicalFunction(pciAddress)
}

// NewPhysicalFunction returns a new api.PhysicalFunction
func (f *Factory) NewPhysicalFunction(pciAddress string) (api.PhysicalFunction, error) {
	return f.newPhysicalFunction(pciAddress)
}

func (f *Factory) newPhysicalFunction(pciAddress string) (*PhysicalFunction, error) {
	var bdfPCIAddress string
	switch {
	case validLongPCIAddr.MatchString(pciAddress):
		bdfPCIAddress = pciAddress
	case validShortPCIAddr.MatchString(pciAddress):
		bdfPCIAddress = bdfDomain + pciAddress
	default:
		return nil, errors.Errorf("invalid PCI address format: %v", pciAddress)
	}

	pciDevicePath := filepath.Join(f.paths.pciDevices, bdfPCIAddress)
	if !isFileExists(pciDevicePath) {
		return nil, errors.Errorf("PCI device doesn't exist: %v", bdfPCIAddress)
	}

	if !isFileExists(filepath.Join(pciDevicePath, totalVfFile)) {
		return nil, errors.Errorf("PCI device is not SR-IOV capable: %v", bdfPCIAddress)
	}

	return &PhysicalFunction{
		Function{
			address: bdfPCIAddress,
			paths:   f.paths,
		},
	}, nil
}
