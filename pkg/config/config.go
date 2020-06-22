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

// Package config define reading settings from file
package config

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
)

// SRIOvTarget contains mac address and additional information
type SRIOvTarget struct {
	MACAddress string            `yaml:"macAddress"`
	Labels     map[string]string `yaml:"labels"` // A set of labels, Endpoint could choose address by.
}

// PCIDevice contains config for each device and corresponding mac address on endpoint side
type PCIDevice struct {
	PCIAddress string            `yaml:"pciAddress"`
	Capability string            `yaml:"capability"`
	Labels     map[string]string `yaml:"labels"` // A set of labels, Endpoint could choose address by.

	Target *SRIOvTarget `yaml:"target"`
}

// ResourceDomain contains host information, name and list of corresponding pci devices
type ResourceDomain struct {
	HostName   string      `yaml:"hostName"`
	PCIDevices []PCIDevice `yaml:"pciDevices"`
}

// ResourceEndpoint contains list of endpoint configuration for each host
type ResourceEndpoint struct {
	Domains []ResourceDomain `yaml:"domains"`
}

// ReadEndpoint reads and parses endpoint config by provided configuration file path
func ReadEndpoint(ctx context.Context, configFile string) (*ResourceEndpoint, error) {
	resources := &ResourceEndpoint{}

	rawBytes, err := ioutil.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return nil, errors.Wrapf(err, "error reading file %s", configFile)
	}

	if err = yaml.Unmarshal(rawBytes, resources); err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling raw bytes")
	}

	log.Entry(ctx).Infof("raw ResourceList: %s", rawBytes)
	log.Entry(ctx).Infof("unmarshalled ResourceList: %+v", resources.Domains)

	return resources, nil
}
