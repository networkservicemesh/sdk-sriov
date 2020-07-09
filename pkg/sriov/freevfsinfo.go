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
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// FreeVirtualFunctionsInfoKey is the key for StateConfig
// TODO move to the API repo
const FreeVirtualFunctionsInfoKey string = "FreeVFs"

// FreeVirtualFunctionsInfo contains information about number of free virtual functions per physical function keyed
// by its PCI address for specified host
type FreeVirtualFunctionsInfo struct {
	FreeVirtualFunctions map[string]int `yaml:"free_vfs"`
}

// Marshall converts FreeVirtualFunctionsInfo to yaml representation
func (f *FreeVirtualFunctionsInfo) Marshall() (string, error) {
	data, err := yaml.Marshal(f)
	if err != nil {
		return "", errors.Wrapf(err, "error marshaling FreeVirtualFunctions: %+v", f)
	}
	strConfig := string(data)

	return strConfig, nil
}

// ParseVirtualFunctionsInfo converts yaml representation of FreeVirtualFunctionsInfo to the golang structure
func ParseVirtualFunctionsInfo(config string) (*FreeVirtualFunctionsInfo, error) {
	stateConfig := &FreeVirtualFunctionsInfo{}

	rawBytes := []byte(config)
	if err := yaml.Unmarshal(rawBytes, stateConfig); err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling raw bytes %s", rawBytes)
	}

	return stateConfig, nil
}
