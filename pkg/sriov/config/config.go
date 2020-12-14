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

// Package config provides SR-IOV config
package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/tools/yamlhelper"
)

// Config contains list of available physical functions
type Config struct {
	PhysicalFunctions map[string]*PhysicalFunction `yaml:"physicalFunctions"`
}

func (c *Config) String() string {
	sb := &strings.Builder{}
	_, _ = sb.WriteString("&{")

	_, _ = sb.WriteString("PhysicalFunctions:map[")
	var strs []string
	for k, physicalFunction := range c.PhysicalFunctions {
		strs = append(strs, fmt.Sprintf("%s:%+v", k, physicalFunction))
	}
	_, _ = sb.WriteString(strings.Join(strs, " "))
	_, _ = sb.WriteString("]")

	_, _ = sb.WriteString("}")
	return sb.String()
}

// PhysicalFunction contains physical function capabilities, available services domains and virtual functions
type PhysicalFunction struct {
	PFKernelDriver   string             `yaml:"pfKernelDriver"`
	VFKernelDriver   string             `yaml:"vfKernelDriver"`
	Capabilities     []string           `yaml:"capabilities"`
	ServiceDomains   []string           `yaml:"serviceDomains"`
	VirtualFunctions []*VirtualFunction `yaml:"virtualFunctions"`
}

func (pf *PhysicalFunction) String() string {
	sb := &strings.Builder{}
	_, _ = sb.WriteString("&{")

	_, _ = sb.WriteString("PFKernelDriver:")
	_, _ = sb.WriteString(pf.PFKernelDriver)

	_, _ = sb.WriteString(" VFKernelDriver:")
	_, _ = sb.WriteString(pf.VFKernelDriver)

	_, _ = sb.WriteString(" Capabilities:[")
	_, _ = sb.WriteString(strings.Join(pf.Capabilities, " "))
	_, _ = sb.WriteString("]")

	_, _ = sb.WriteString(" ServiceDomains:[")
	_, _ = sb.WriteString(strings.Join(pf.ServiceDomains, " "))
	_, _ = sb.WriteString("]")

	_, _ = sb.WriteString(" VirtualFunctions:[")
	var strs []string
	for _, virtualFunction := range pf.VirtualFunctions {
		strs = append(strs, fmt.Sprintf("%+v", virtualFunction))
	}
	_, _ = sb.WriteString(strings.Join(strs, " "))
	_, _ = sb.WriteString("]")

	_, _ = sb.WriteString("}")
	return sb.String()
}

// VirtualFunction contains
type VirtualFunction struct {
	Address    string `yaml:"address"`
	IOMMUGroup uint   `yaml:"iommuGroup"`
}

// ReadConfig reads configuration from file
func ReadConfig(ctx context.Context, configFile string) (*Config, error) {
	logEntry := log.Entry(ctx).WithField("Config", "ReadConfig")

	cfg := &Config{}
	if err := yamlhelper.UnmarshalFile(configFile, cfg); err != nil {
		return nil, err
	}

	for pciAddr, pfCfg := range cfg.PhysicalFunctions {
		if pfCfg.PFKernelDriver == "" {
			return nil, errors.Errorf("%s has no PFKernelDriver set", pciAddr)
		}
		if pfCfg.VFKernelDriver == "" {
			return nil, errors.Errorf("%s has no VFKernelDriver set", pciAddr)
		}
		if len(pfCfg.Capabilities) == 0 {
			return nil, errors.Errorf("%s has no Capabilities set", pciAddr)
		}
		if len(pfCfg.ServiceDomains) == 0 {
			return nil, errors.Errorf("%s has no ServiceDomains set", pciAddr)
		}
	}

	logEntry.Infof("unmarshalled Config: %+v", cfg)

	return cfg, nil
}
