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

package pcifunction

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"

	api "github.com/networkservicemesh/sdk-sriov/pkg/tools/api/pcifunction"
)

const (
	totalVfFile            = "sriov_totalvfs"
	configuredVfFile       = "sriov_numvfs"
	virtualFunctionPattern = "virtfn*"
)

// PhysicalFunction describes Linux PCI physical function
type PhysicalFunction struct {
	Function
}

// GetVirtualFunctionsCapacity returns count of virtual functions that can be created for the pf
func (pf *PhysicalFunction) GetVirtualFunctionsCapacity() (uint, error) {
	return readUintFromFile(filepath.Join(pf.paths.pciDevices, pf.address, totalVfFile))
}

// CreateVirtualFunctions initializes virtual functions for the pf
// NOTE: should fail if virtual functions are already exist
func (pf *PhysicalFunction) CreateVirtualFunctions(vfCount uint) error {
	configuredVfFilePath := filepath.Join(pf.paths.pciDevices, pf.address, configuredVfFile)
	err := ioutil.WriteFile(configuredVfFilePath, []byte(strconv.Itoa(int(vfCount))), 0)
	if err != nil {
		return errors.Wrapf(err, "failed to create virtual functions for the device: %v", pf.address)
	}

	return nil
}

// GetVirtualFunctions returns all virtual functions discovered for the pf
func (pf *PhysicalFunction) GetVirtualFunctions() ([]api.BindablePCIFunction, error) {
	vfDirs, err := filepath.Glob(filepath.Join(pf.paths.pciDevices, pf.address, virtualFunctionPattern))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find virtual function directories for the device: %v", pf.address)
	}

	var pcifs []api.BindablePCIFunction
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				pcifs = append(pcifs, &Function{
					address: filepath.Base(linkName),
					paths:   pf.paths,
				})
			}
		}
	}

	return pcifs, nil
}
