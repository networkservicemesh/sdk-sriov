// Copyright (c) 2022 Cisco and/or its affiliates.
//
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

// Package resource provides a resource pool for SR-IOV PCI virtual functions
package resource

import (
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
)

// TokenPool is a token.Pool interface
type TokenPool interface {
	Find(id string) (string, error)
	Use(id string, names []string) error
	StopUsing(id string) error
}

// Pool manages host SR-IOV state
// WARNING: it is thread unsafe - if you want to use it concurrently, use some synchronization outside
type Pool struct {
	physicalFunctions map[string]*physicalFunction
	virtualFunctions  map[string]*virtualFunction
	tokens            map[string]*virtualFunction
	iommuGroups       map[uint]sriov.DriverType
	tokenPool         TokenPool
}

type physicalFunction struct {
	tokenNames       map[string]struct{}
	virtualFunctions map[uint][]*virtualFunction
	freeVFsCount     int
}

type virtualFunction struct {
	pciAddr    string
	pfPCIAddr  string
	iommuGroup uint
	tokenID    string
}

// NewPool returns a new Pool
func NewPool(tokenPool TokenPool, cfg *config.Config) *Pool {
	p := &Pool{
		physicalFunctions: map[string]*physicalFunction{},
		virtualFunctions:  map[string]*virtualFunction{},
		tokens:            map[string]*virtualFunction{},
		iommuGroups:       map[uint]sriov.DriverType{},
		tokenPool:         tokenPool,
	}

	for pfPCIAddr, pFun := range cfg.PhysicalFunctions {
		pf := &physicalFunction{
			tokenNames:       map[string]struct{}{},
			virtualFunctions: map[uint][]*virtualFunction{},
			freeVFsCount:     len(pFun.VirtualFunctions),
		}
		p.physicalFunctions[pfPCIAddr] = pf

		for _, serviceDomain := range pFun.ServiceDomains {
			for _, capability := range pFun.Capabilities {
				pf.tokenNames[path.Join(serviceDomain, capability)] = struct{}{}
			}
		}

		for _, vFun := range pFun.VirtualFunctions {
			vf := &virtualFunction{
				pciAddr:    vFun.Address,
				pfPCIAddr:  pfPCIAddr,
				iommuGroup: vFun.IOMMUGroup,
			}
			p.virtualFunctions[vFun.Address] = vf

			pf.virtualFunctions[vFun.IOMMUGroup] = append(pf.virtualFunctions[vFun.IOMMUGroup], vf)
			p.iommuGroups[vFun.IOMMUGroup] = sriov.NoDriver
		}
	}

	return p
}

// Select selects a virtual function for the given driver type and marks it as "in-use"
func (p *Pool) Select(tokenID string, driverType sriov.DriverType) (string, error) {
	switch vf, err := p.trySelected(tokenID, driverType); {
	case err != nil:
		return "", err
	case vf != nil:
		return vf.pciAddr, nil
	}

	tokenName, err := p.tokenPool.Find(tokenID)
	if err != nil {
		return "", err
	}

	vfs := p.find(driverType, tokenName)
	if len(vfs) == 0 {
		return "", errors.Errorf("no free VF for the driver type: %v", driverType)
	}

	sort.Slice(vfs, func(i, k int) bool {
		//revive:disable:var-naming
		iIg := p.iommuGroups[vfs[i].iommuGroup]
		kIg := p.iommuGroups[vfs[k].iommuGroup]
		iPF := p.physicalFunctions[vfs[i].pfPCIAddr]
		kPF := p.physicalFunctions[vfs[k].pfPCIAddr]
		//revive:enable:var-naming
		switch {
		case iIg == driverType && kIg == sriov.NoDriver:
			return true
		case iIg == sriov.NoDriver && kIg == driverType:
			return false
		case iPF.freeVFsCount > kPF.freeVFsCount:
			return true
		case iPF.freeVFsCount < kPF.freeVFsCount:
			return false
		default:
			// we need this additional comparison to make sort deterministic
			return strings.Compare(vfs[i].pciAddr, vfs[k].pciAddr) < 0
		}
	})

	if err := p.selectVF(vfs[0], tokenID, driverType); err != nil {
		return "", err
	}

	return vfs[0].pciAddr, nil
}

func (p *Pool) trySelected(tokenID string, driverType sriov.DriverType) (*virtualFunction, error) {
	if vf, ok := p.tokens[tokenID]; ok {
		if p.iommuGroups[vf.iommuGroup] != driverType {
			return nil, p.Free(vf.pciAddr)
		}
		return vf, nil
	}
	return nil, nil
}

func (p *Pool) find(driverType sriov.DriverType, tokenName string) []*virtualFunction {
	var virtualFunctions []*virtualFunction
	for _, pf := range p.physicalFunctions {
		if _, ok := pf.tokenNames[tokenName]; ok {
			for iommuGroup, vfs := range pf.virtualFunctions {
				if ig := p.iommuGroups[iommuGroup]; ig == sriov.NoDriver || ig == driverType {
					for _, vf := range vfs {
						if vf.tokenID == "" {
							virtualFunctions = append(virtualFunctions, vf)
						}
					}
				}
			}
		}
	}
	return virtualFunctions
}

func (p *Pool) selectVF(vf *virtualFunction, tokenID string, driverType sriov.DriverType) error {
	var tokenNames []string
	for tokenName := range p.physicalFunctions[vf.pfPCIAddr].tokenNames {
		tokenNames = append(tokenNames, tokenName)
	}
	if err := p.tokenPool.Use(tokenID, tokenNames); err != nil {
		return err
	}

	p.tokens[tokenID] = vf
	vf.tokenID = tokenID

	p.physicalFunctions[vf.pfPCIAddr].freeVFsCount--
	p.iommuGroups[vf.iommuGroup] = driverType

	return nil
}

// Free marks given virtual function as "free" and binds it to the "NoDriver" driver type
func (p *Pool) Free(vfPCIAddr string) error {
	vf, ok := p.virtualFunctions[vfPCIAddr]
	if !ok {
		return errors.Errorf("VF doesn't exist: %v", vfPCIAddr)
	}

	if vf.tokenID == "" {
		return errors.Errorf("trying to free not selected VF: %v", vf.pciAddr)
	}
	if err := p.tokenPool.StopUsing(vf.tokenID); err != nil {
		return err
	}
	delete(p.tokens, vf.tokenID)
	vf.tokenID = ""

	p.physicalFunctions[vf.pfPCIAddr].freeVFsCount++

	for _, pf := range p.physicalFunctions {
		if vffs, ok := pf.virtualFunctions[vf.iommuGroup]; ok {
			for _, vff := range vffs {
				if vff.tokenID != "" {
					return nil
				}
			}
		}
	}
	p.iommuGroups[vf.iommuGroup] = sriov.NoDriver

	return nil
}
