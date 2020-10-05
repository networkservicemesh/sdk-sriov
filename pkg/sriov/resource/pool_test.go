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

package resource_test

import (
	"context"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/resource"
)

const (
	configFileName  = "config.yml"
	service1        = "service-1"
	service2        = "service-2"
	capabilityIntel = "intel"
	capability10G   = "10G"
	vf11PciAddr     = "0000:01:00.1"
	vf21PciAddr     = "0000:02:00.1"
	vf22PciAddr     = "0000:02:00.2"
	vf31PciAddr     = "0000:03:00.1"
)

func TestPool_Select_Service(t *testing.T) {
	tokenPool := &tokenPoolStub{
		tokens: map[string]string{
			"1": path.Join(service1, capabilityIntel),
		},
	}

	cfg, err := config.ReadConfig(context.TODO(), configFileName)
	require.NoError(t, err)

	p := resource.NewPool(tokenPool, cfg)

	vfPCIAddr, err := p.Select("1", sriov.VFIOPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPCIAddr)
}

func TestPool_Select_Capability(t *testing.T) {
	tokenPool := &tokenPoolStub{
		tokens: map[string]string{
			"1": path.Join(service2, capability10G),
		},
	}

	cfg, err := config.ReadConfig(context.TODO(), configFileName)
	require.NoError(t, err)

	p := resource.NewPool(tokenPool, cfg)

	vfPCIAddr, err := p.Select("1", sriov.VFIOPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf21PciAddr, vfPCIAddr)
}

func TestPool_Select_DriverType(t *testing.T) {
	tokenPool := &tokenPoolStub{
		tokens: map[string]string{
			"1": path.Join(service1, capabilityIntel),
			"2": path.Join(service2, capabilityIntel),
		},
	}

	cfg, err := config.ReadConfig(context.TODO(), configFileName)
	require.NoError(t, err)

	p := resource.NewPool(tokenPool, cfg)

	vfPCIAddr, err := p.Select("1", sriov.VFIOPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPCIAddr)

	vfPCIAddr, err = p.Select("2", sriov.KernelDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf22PciAddr, vfPCIAddr)
}

func TestPool_Select_FreeVFsCount(t *testing.T) {
	tokenPool := &tokenPoolStub{
		tokens: map[string]string{
			"1": path.Join(service2, capabilityIntel),
		},
	}

	cfg, err := config.ReadConfig(context.TODO(), configFileName)
	require.NoError(t, err)

	p := resource.NewPool(tokenPool, cfg)

	vfPCIAddr, err := p.Select("1", sriov.VFIOPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf31PciAddr, vfPCIAddr)
}

func TestPool_Free(t *testing.T) {
	tokenPool := &tokenPoolStub{
		tokens: map[string]string{
			"1": path.Join(service1, capabilityIntel),
		},
	}

	cfg, err := config.ReadConfig(context.TODO(), configFileName)
	require.NoError(t, err)

	p := resource.NewPool(tokenPool, cfg)

	vfPCIAddr, err := p.Select("1", sriov.VFIOPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPCIAddr)

	err = p.Free(vfPCIAddr)
	assert.Nil(t, err)

	vfPCIAddr, err = p.Select("1", sriov.VFIOPCIDriver)
	assert.Nil(t, err)
	assert.Equal(t, vf11PciAddr, vfPCIAddr)
}

type tokenPoolStub struct {
	tokens map[string]string
}

func (tp *tokenPoolStub) Find(id string) (string, error) {
	if tokenName, ok := tp.tokens[id]; ok {
		return tokenName, nil
	}
	return "", errors.New("invalid token ID")
}

func (tp *tokenPoolStub) Use(id string, _ []string) error {
	if _, ok := tp.tokens[id]; ok {
		return nil
	}
	return errors.New("invalid token ID")
}

func (tp *tokenPoolStub) StopUsing(id string) error {
	if _, ok := tp.tokens[id]; ok {
		return nil
	}
	return errors.New("invalid token ID")
}
