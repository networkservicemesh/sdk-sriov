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

package resourcepool_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/resourcepool"
)

const (
	configFileName                  = "config.yml"
	hostName                        = "service1.example.com"
	pf1PciAddr                      = "0000:01:00.0"
	pf2PciAddr                      = "0000:02:00.0"
	pf3PciAddr                      = "0000:03:00.0"
	pf1Capability  sriov.Capability = "10G"
	pf2Capability  sriov.Capability = "20G"
	pf3Capability  sriov.Capability = "30G"
)

// TestReadConfigFile test reading a SRIOV config file
func TestReadConfigFile(t *testing.T) {
	config, err := resourcepool.ReadConfig(context.Background(), configFileName)
	assert.Nil(t, err)
	assert.Equal(t, &resourcepool.Config{
		HostName: hostName,
		PhysicalFunctions: map[string]sriov.Capability{
			pf1PciAddr: pf1Capability,
			pf2PciAddr: pf2Capability,
			pf3PciAddr: pf3Capability,
		},
	}, config)
}
