// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package pcifunction

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

const (
	netInterfacesPath = "net"
	iommuGroup        = "iommu_group"
	boundDriverPath   = "driver"
	bindDriverPath    = "bind"
	unbindDriverPath  = "unbind"
)

// Function describes Linux PCI function
type Function struct {
	address        string
	pciDevicesPath string
	pciDriversPath string
}

// GetPCIAddress returns f PCI address
func (f *Function) GetPCIAddress() string {
	return f.address
}

// GetNetInterfaceName returns f net interface name
func (f *Function) GetNetInterfaceName() (string, error) {
	fInfos, err := ioutil.ReadDir(f.withDevicePath(netInterfacesPath))
	if err != nil {
		return "", errors.Wrapf(err, "failed to read net directory for the device: %v", f.address)
	}

	var ifNames []string
	for _, fInfo := range fInfos {
		ifNames = append(ifNames, fInfo.Name())
	}

	switch len(ifNames) {
	case 0:
		return "", errors.Errorf("no interfaces found for the device: %v - %+v", f.address, ifNames)
	case 1:
		return ifNames[0], nil
	default:
		return "", errors.Errorf("found multiple interfaces for the device: %v - %+v", f.address, ifNames)
	}
}

// GetIOMMUGroup returns f IOMMU group id
func (f *Function) GetIOMMUGroup() (uint, error) {
	stringIOMMUGroup, err := evalSymlinkAndGetBaseName(f.withDevicePath(iommuGroup))
	if err != nil {
		return 0, err
	}

	iommuGroup, _ := strconv.Atoi(stringIOMMUGroup)

	return uint(iommuGroup), nil
}

// GetBoundDriver returns driver name that is bound to f, if no driver bound, returns ""
func (f *Function) GetBoundDriver() (string, error) {
	if !isFileExists(f.withDevicePath(boundDriverPath)) {
		return "", nil
	}

	driver, err := evalSymlinkAndGetBaseName(f.withDevicePath(boundDriverPath))
	if err != nil {
		return "", err
	}

	return driver, nil
}

// BindDriver unbinds currently bound driver and binds the given driver to f
func (f *Function) BindDriver(driver string) error {
	switch boundDriver, err := f.GetBoundDriver(); {
	case err != nil:
		return err
	case boundDriver == driver:
		return nil
	case boundDriver != "":
		unbindPath := f.withDevicePath(boundDriverPath, unbindDriverPath)
		if err := ioutil.WriteFile(unbindPath, []byte(f.address), 0); err != nil {
			return errors.Wrapf(err, "failed to unbind driver from the device: %v", f.address)
		}
	}

	// For some reasons write to the driver/bind file fails but binds the driver to the PCI function
	// so we ignore error and simply compare the bound driver with the given one
	bindPath := filepath.Join(f.pciDriversPath, driver, bindDriverPath)
	err := ioutil.WriteFile(bindPath, []byte(f.address), 0)
	if boundDriver, _ := f.GetBoundDriver(); boundDriver != driver {
		return errors.Wrapf(err, "failed to bind the driver to the device: %v %v", f.address, driver)
	}

	return nil
}

func (f *Function) withDevicePath(elem ...string) string {
	return path.Join(append([]string{f.pciDevicesPath, f.address}, elem...)...)
}
