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

package resourcepool

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	types "github.com/networkservicemesh/sdk-sriov/pkg/sriov/types/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/yamlhelper"
)

// Config contains list of available physical functions and their capabilities
type Config struct {
	HostName          string                      `yaml:"hostName"`
	PhysicalFunctions map[string]types.Capability `yaml:"physicalFunctions"`
}

// ReadConfig reads configuration from file
func ReadConfig(ctx context.Context, configFile string) (*Config, error) {
	logEntry := log.Entry(ctx).WithField("Config", "ReadConfig")

	config := &Config{}
	if err := yamlhelper.UnmarshalFile(configFile, config); err != nil {
		return nil, err
	}

	valid := true
	for _, capability := range config.PhysicalFunctions {
		if err := capability.Validate(); err != nil {
			logEntry.Error(err.Error())
			valid = false
		}
	}
	if !valid {
		return nil, errors.Errorf("error validating data types for %v", config)
	}

	logEntry.Infof("unmarshalled Config: %+v", config)

	return config, nil
}
