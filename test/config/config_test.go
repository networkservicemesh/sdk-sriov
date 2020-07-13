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

package config_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

const (
	hostName   = "service1.example.com"
	pciAddress = "0000:01:00:0"
	capability = "10G"
	macAddress = "00:0a:95:9d:68:16"

	configFileName = "config.yml"
)

// TestReadConfigFile test reading a endpoint config file
func TestReadConfigFile(t *testing.T) {
	configList, _ := sriov.ReadConfig(context.Background(), configFileName)
	assert.NotNil(t, configList)
	pciDevices := configList.Domains[hostName]
	assert.NotNil(t, pciDevices)
	pciDevice := pciDevices[0]
	assert.Equal(t, pciAddress, pciDevice.PCIAddress)
	assert.Equal(t, capability, pciDevice.Capability)
	assert.Equal(t, macAddress, pciDevice.Target.MACAddress)
}
