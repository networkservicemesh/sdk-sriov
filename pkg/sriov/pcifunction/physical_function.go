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

// Package pcifunction provides classes for linux PCI functions API
package pcifunction

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

// TODO: add unit tests with sriovtest.FileAPI

const (
	bdfDomain              = "0000:"
	totalVfFile            = "sriov_totalvfs"
	configuredVfFile       = "sriov_numvfs"
	virtualFunctionPattern = "virtfn*"
)

var (
	validLongPCIAddr  = regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]{1}$`)
	validShortPCIAddr = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]{1}$`)
)

// PhysicalFunction describes Linux PCI physical function
type PhysicalFunction struct {
	Function
}

// NewPhysicalFunction returns a new PhysicalFunction
func NewPhysicalFunction(pciAddress, pciDevicesPath, pciDriversPath string) (*PhysicalFunction, error) {
	var bdfPCIAddress string
	switch {
	case validLongPCIAddr.MatchString(pciAddress):
		bdfPCIAddress = pciAddress
	case validShortPCIAddr.MatchString(pciAddress):
		bdfPCIAddress = bdfDomain + pciAddress
	default:
		return nil, errors.Errorf("invalid PCI address format: %v", pciAddress)
	}

	pciDevicePath := filepath.Join(pciDevicesPath, bdfPCIAddress)
	if !isFileExists(pciDevicePath) {
		return nil, errors.Errorf("PCI device doesn't exist: %v", bdfPCIAddress)
	}

	if !isFileExists(filepath.Join(pciDevicePath, totalVfFile)) {
		return nil, errors.Errorf("PCI device is not SR-IOV capable: %v", bdfPCIAddress)
	}

	f, err := newFunction(bdfPCIAddress, pciDevicesPath, pciDriversPath)
	if err != nil {
		return nil, err
	}

	return &PhysicalFunction{
		Function: *f,
	}, nil
}

// GetVirtualFunctionsCapacity returns count of virtual functions that can be created for the pf
func (pf *PhysicalFunction) GetVirtualFunctionsCapacity() (uint, error) {
	return readUintFromFile(filepath.Join(pf.pciDevicesPath, pf.address, totalVfFile))
}

// CreateVirtualFunctions initializes virtual functions for the pf
// NOTE: should fail if virtual functions are already exist
func (pf *PhysicalFunction) CreateVirtualFunctions(vfCount uint) error {
	configuredVfFilePath := filepath.Join(pf.pciDevicesPath, pf.address, configuredVfFile)
	err := ioutil.WriteFile(configuredVfFilePath, []byte(strconv.Itoa(int(vfCount))), 0)
	if err != nil {
		return errors.Wrapf(err, "failed to create virtual functions for the device: %v", pf.address)
	}

	return nil
}

// GetVirtualFunctions returns all virtual functions discovered for the pf
func (pf *PhysicalFunction) GetVirtualFunctions() ([]*Function, error) {
	vfDirs, err := filepath.Glob(filepath.Join(pf.pciDevicesPath, pf.address, virtualFunctionPattern))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find virtual function directories for the device: %v", pf.address)
	}

	var fs []*Function
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				if f, err := newFunction(filepath.Base(linkName), pf.pciDevicesPath, pf.pciDriversPath); err == nil {
					fs = append(fs, f)
				}
			}
		}
	}

	return fs, nil
}
