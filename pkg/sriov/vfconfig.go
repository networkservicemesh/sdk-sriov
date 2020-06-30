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

package sriov

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
)

// VirtualFunctionsStateConfigKey is the key for StateConfig
// TODO move to the API repo
const VirtualFunctionsStateConfigKey string = "VirtualFunctionStateConfig"

// VirtualFunctionsStateConfig contains information about number of free virtual functions per physical function keyed
// by its PCI address
type VirtualFunctionsStateConfig struct {
	Config map[string]int `yaml:"config"`
}

// MarshallStateConfig converts VirtualFunctionsStateConfig to yaml representation
func MarshallStateConfig(ctx context.Context, config *VirtualFunctionsStateConfig) (string, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", errors.Wrapf(err, "error marshaling Config: %+v", config)
	}

	strConfig := string(data)

	log.Entry(ctx).Infof("Config: %+v", config)
	log.Entry(ctx).Infof("marshaled Config: %s", strConfig)

	return strConfig, nil
}

// UnmarshallStateConfig converts yaml representation of VirtualFunctionsStateConfig to go structure
func UnmarshallStateConfig(ctx context.Context, config string) (*VirtualFunctionsStateConfig, error) {
	stateConfig := &VirtualFunctionsStateConfig{}

	rawBytes := []byte(config)
	if err := yaml.Unmarshal(rawBytes, stateConfig); err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling raw bytes %s", rawBytes)
	}

	log.Entry(ctx).Infof("raw Config: %s", rawBytes)
	log.Entry(ctx).Infof("unmarshalled Config: %+v", stateConfig.Config)

	return stateConfig, nil
}
