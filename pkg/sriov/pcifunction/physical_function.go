// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// TODO: add unit tests with sriovtest.FileAPI

const (
	bdfDomain             = "0000:"
	totalVFFile           = "sriov_totalvfs"
	configuredVFFile      = "sriov_numvfs"
	virtualFunctionPrefix = "virtfn"
)

var (
	validLongPCIAddr  = regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]{1}$`)
	validShortPCIAddr = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]{1}$`)
)

// PhysicalFunction describes Linux PCI physical function
type PhysicalFunction struct {
	virtualFunctions []*Function

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

	if !isFileExists(filepath.Join(pciDevicePath, totalVFFile)) {
		return nil, errors.Errorf("PCI device is not SR-IOV capable: %v", bdfPCIAddress)
	}

	pf := &PhysicalFunction{
		Function: Function{
			address:        pciAddress,
			pciDevicesPath: pciDevicesPath,
			pciDriversPath: pciDriversPath,
		},
	}
	if err := pf.createVirtualFunctions(); err != nil {
		return nil, err
	}
	if err := pf.loadVirtualFunctions(); err != nil {
		return nil, err
	}
	return pf, nil
}

// GetVirtualFunctions returns pf virtual functions
func (pf *PhysicalFunction) GetVirtualFunctions() []*Function {
	vfs := make([]*Function, len(pf.virtualFunctions))
	copy(vfs, pf.virtualFunctions)
	return vfs
}

func (pf *PhysicalFunction) createVirtualFunctions() error {
	switch vfsCount, err := readUintFromFile(pf.withDevicePath(configuredVFFile)); {
	case err != nil:
		return errors.Wrapf(err, "failed to get configured VFs number for the PCI device: %v", pf.address)
	case vfsCount > 0:
		return nil
	}

	vfsCount, err := ioutil.ReadFile(pf.withDevicePath(totalVFFile))
	if err != nil {
		return errors.Wrapf(err, "failed to get available VFs number for the PCI device: %v", pf.address)
	}

	err = ioutil.WriteFile(pf.withDevicePath(configuredVFFile), vfsCount, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to create VFs for the PCI device: %v", pf.address)
	}

	return nil
}

func (pf *PhysicalFunction) loadVirtualFunctions() error {
	vfDirs, err := filepath.Glob(pf.withDevicePath(virtualFunctionPrefix + "*"))
	if err != nil {
		return errors.Wrapf(err, "failed to find virtual function directories for the device: %v", pf.address)
	}

	sort.Slice(vfDirs, func(i, k int) bool {
		leftVFNum, _ := strconv.Atoi(strings.TrimPrefix(vfDirs[i], virtualFunctionPrefix))
		rightVFNum, _ := strconv.Atoi(strings.TrimPrefix(vfDirs[k], virtualFunctionPrefix))
		return leftVFNum < rightVFNum
	})

	for _, vfDir := range vfDirs {
		vfDirInfo, err := os.Lstat(vfDir)
		if err != nil || vfDirInfo.Mode()&os.ModeSymlink == 0 {
			return errors.Wrapf(err, "invalid virtual function directory: %v", vfDir)
		}

		linkName, err := filepath.EvalSymlinks(vfDir)
		if err != nil {
			return errors.Wrapf(err, "invalid virtual function directory: %v", vfDir)
		}

		pf.virtualFunctions = append(pf.virtualFunctions, &Function{
			address:        filepath.Base(linkName),
			pciDevicesPath: pf.pciDevicesPath,
			pciDriversPath: pf.pciDriversPath,
		})
	}
	return nil
}
