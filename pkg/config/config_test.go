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
	registryDomainName1 = "domain1"
	capability1         = "capability1"
	devicePciAddress1   = "pciAddr1"
	connectedToPort1    = "macAddr1"

	configFileName = "config.yml"
)

func TestReadConfigFile(t *testing.T) {
	configList, _ := config.ReadConfig(configFileName)
	assert.NotNil(t, configList)
	resConfig1 := configList.ResourceList[0]
	assert.NotNil(t, resConfig1)
	assert.Equal(t, resConfig1.RegistryDomainName, registryDomainName1)
	assert.Equal(t, resConfig1.Capability, capability1)
	assert.Equal(t, resConfig1.DevicePciAddress, devicePciAddress1)
	assert.Equal(t, resConfig1.ConnectedToPort, connectedToPort1)
}
