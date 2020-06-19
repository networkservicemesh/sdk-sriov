// Copyright (c) 2020 Cisco and/or its affiliates.
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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk-sriov/pkg/config"
)

const (
	networkServiceName = "networkServiceName"
	sourceHostName     = "example.org"
	sourcePCIAddress   = "0000:01:00:0"
	capability         = "10G"
	targetMACAddress   = "00:0a:95:9d:68:16"

	configFileName = "config.yml"
)

// TestReadConfigFile test reading a endpoint config file
func TestReadEndpointConfigFile(t *testing.T) {
	configList, _ := config.ReadConfig(configFileName)
	assert.NotNil(t, configList)
	resConfig := configList.ResourceList[0]
	assert.NotNil(t, resConfig)
	assert.Equal(t, resConfig.NetworkServiceName, networkServiceName)
	assert.Equal(t, resConfig.SourceHostName, sourceHostName)
	assert.Equal(t, resConfig.SourcePCIAddress, sourcePCIAddress)
	assert.Equal(t, resConfig.Capability, capability)
	assert.Equal(t, resConfig.TargetMACAddress, targetMACAddress)
}
