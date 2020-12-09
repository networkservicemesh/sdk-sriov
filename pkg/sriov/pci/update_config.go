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

package pci

import (
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/pcifunction"
)

// UpdateConfig updates config with virtual functions
func UpdateConfig(pciDevicesPath, pciDriversPath string, cfg *config.Config) error {
	for pfPCIAddr, pfCfg := range cfg.PhysicalFunctions {
		pf, err := pcifunction.NewPhysicalFunction(pfPCIAddr, pciDevicesPath, pciDriversPath)
		if err != nil {
			return err
		}

		for _, vf := range pf.GetVirtualFunctions() {
			iommuGroup, err := vf.GetIOMMUGroup()
			if err != nil {
				return err
			}

			pfCfg.VirtualFunctions = append(pfCfg.VirtualFunctions, &config.VirtualFunction{
				Address:    vf.GetPCIAddress(),
				IOMMUGroup: iommuGroup,
			})
		}
	}
	return nil
}
