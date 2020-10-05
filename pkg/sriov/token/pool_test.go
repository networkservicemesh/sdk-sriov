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

package token_test

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/token"
)

const (
	configFileName  = "config.yml"
	service1        = "service-1"
	service2        = "service-2"
	capabilityIntel = "intel"
	capability10G   = "10G"
	capability20G   = "20G"
)

func TestPool_Tokens(t *testing.T) {
	cfg, err := config.ReadConfig(context.TODO(), configFileName)
	require.NoError(t, err)

	p := token.NewPool(cfg)

	tokens := p.Tokens()
	require.Equal(t, 5, len(tokens))
	require.Equal(t, 4, countTrue(tokens[path.Join(service1, capabilityIntel)]))
	require.Equal(t, 1, countTrue(tokens[path.Join(service1, capability10G)]))
	require.Equal(t, 3, countTrue(tokens[path.Join(service1, capability20G)]))
	require.Equal(t, 3, countTrue(tokens[path.Join(service2, capabilityIntel)]))
	require.Equal(t, 3, countTrue(tokens[path.Join(service2, capability20G)]))
}

func TestPool_Use(t *testing.T) {
	cfg, err := config.ReadConfig(context.TODO(), configFileName)
	require.NoError(t, err)

	p := token.NewPool(cfg)

	var tokenID string
	for id := range p.Tokens()[path.Join(service2, capability20G)] {
		err = p.Use(id, []string{
			path.Join(service1, capabilityIntel),
			path.Join(service1, capability20G),
			path.Join(service2, capabilityIntel),
			path.Join(service2, capability20G),
		})
		require.NoError(t, err)
		tokenID = id
	}

	tokens := p.Tokens()
	require.Equal(t, 5, len(tokens))
	require.Equal(t, 1, countTrue(tokens[path.Join(service1, capabilityIntel)]))
	require.Equal(t, 1, countTrue(tokens[path.Join(service1, capability10G)]))
	require.Equal(t, 0, countTrue(tokens[path.Join(service1, capability20G)]))
	require.Equal(t, 0, countTrue(tokens[path.Join(service2, capabilityIntel)]))
	require.Equal(t, 3, countTrue(tokens[path.Join(service2, capability20G)]))

	err = p.StopUsing(tokenID)
	require.NoError(t, err)

	tokens = p.Tokens()
	require.Equal(t, 5, len(tokens))
	require.Equal(t, 2, countTrue(tokens[path.Join(service1, capabilityIntel)]))
	require.Equal(t, 1, countTrue(tokens[path.Join(service1, capability10G)]))
	require.Equal(t, 1, countTrue(tokens[path.Join(service1, capability20G)]))
	require.Equal(t, 1, countTrue(tokens[path.Join(service2, capabilityIntel)]))
	require.Equal(t, 3, countTrue(tokens[path.Join(service2, capability20G)]))
}

func countTrue(m map[string]bool) (count int) {
	for _, v := range m {
		if v {
			count++
		}
	}
	return count
}
