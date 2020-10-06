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

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
)

const (
	configFileName  = "config.yml"
	pf1PciAddr      = "0000:01:00.0"
	pf2PciAddr      = "0000:02:00.0"
	capabilityIntel = "intel"
	capability10G   = "10G"
	capability20G   = "20G"
	serviceDomain1  = "service.domain.1"
	serviceDomain2  = "service.domain.2"
	vf11PciAddr     = "0000:01:00.1"
	vf12PciAddr     = "0000:01:00.2"
	vf21PciAddr     = "0000:02:00.1"
	vf22PciAddr     = "0000:02:00.2"
	vf23PciAddr     = "0000:02:00.3"
)

func TestReadConfigFile(t *testing.T) {
	cfg, err := config.ReadConfig(context.Background(), configFileName)
	require.NoError(t, err)
	require.Equal(t, &config.Config{
		PhysicalFunctions: map[string]*config.PhysicalFunction{
			pf1PciAddr: {
				Capabilities: []string{
					capabilityIntel,
					capability10G,
				},
				ServiceDomains: []string{
					serviceDomain1,
				},
				VirtualFunctions: map[string]uint{
					vf11PciAddr: 1,
					vf12PciAddr: 2,
				},
			},
			pf2PciAddr: {
				Capabilities: []string{
					capabilityIntel,
					capability20G,
				},
				ServiceDomains: []string{
					serviceDomain1,
					serviceDomain2,
				},
				VirtualFunctions: map[string]uint{
					vf21PciAddr: 1,
					vf22PciAddr: 2,
					vf23PciAddr: 3,
				},
			},
		},
	}, cfg)
}
