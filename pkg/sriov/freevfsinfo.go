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

// MarshallFreeVirtualFunctionsInfo converts FreeVirtualFunctionsInfo to yaml representation
func MarshallFreeVirtualFunctionsInfo(config *FreeVirtualFunctionsInfo) (string, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", errors.Wrapf(err, "error marshaling FreeVirtualFunctions: %+v", config)
	}
	strConfig := string(data)

	return strConfig, nil
}

// UnmarshallFreeVirtualFunctionsInfo converts yaml representation of FreeVirtualFunctionsInfo to the golang structure
func UnmarshallFreeVirtualFunctionsInfo(config string) (*FreeVirtualFunctionsInfo, error) {
	stateConfig := &FreeVirtualFunctionsInfo{}

	rawBytes := []byte(config)
	if err := yaml.Unmarshal(rawBytes, stateConfig); err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling raw bytes %s", rawBytes)
	}

	return stateConfig, nil
}
