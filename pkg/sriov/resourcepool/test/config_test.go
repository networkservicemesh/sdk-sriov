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

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/resourcepool"
)

const (
	pf1PciAddr = "0000:01:00.0"
	pf2PciAddr = "0000:02:00.0"
	pf3PciAddr = "0000:03:00.0"
	pf4PciAddr = "0000:04:00.0"
	pf5PciAddr = "0000:05:00.0"
)

// TestReadConfigFile test reading a SRIOV config file
func TestReadConfigFile(t *testing.T) {
	config, err := resourcepool.ReadConfig(context.Background(), configFileName)
	assert.Nil(t, err)
	assert.Equal(t, &resourcepool.Config{
		PhysicalFunctions: map[string]*resourcepool.PhysicalFunction{
			pf1PciAddr: {
				Capability: pf1Capability,
				Services: []string{
					service1,
				},
			},
			pf2PciAddr: {
				Capability: pf2Capability,
				Services: []string{
					service1,
				},
			},
			pf3PciAddr: {
				Capability: pf3Capability,
				Services: []string{
					service1,
					service2,
				},
			},
			pf4PciAddr: {
				Capability: pf4Capability,
				Services: []string{
					service1,
					service2,
				},
			},
			pf5PciAddr: {
				Capability: pf4Capability,
				Services: []string{
					service1,
					service2,
				},
			},
		},
	}, config)
}
