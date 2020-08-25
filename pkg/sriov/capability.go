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
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

// Capability is a type for PCI function capability
type Capability string

var validCapability = regexp.MustCompile(`^([1-9][0-9]*)([GMK]??)b??$`)

// Validate validates Capability string
func (c Capability) Validate() error {
	if validCapability.MatchString(string(c)) {
		return nil
	}
	return errors.Errorf("PCI capability %v expected to be in format: %v", c, validCapability)
}

// Compare compares capabilities
func (c Capability) Compare(other Capability) int {
	return c.toBytes() - other.toBytes()
}

func (c Capability) toBytes() int {
	parsed := validCapability.FindStringSubmatch(string(c))
	bytes, _ := strconv.Atoi(parsed[1])
	switch parsed[2] {
	case "G":
		bytes *= 1024
		fallthrough
	case "M":
		bytes *= 1024
		fallthrough
	case "K":
		bytes *= 1024
	}
	return bytes
}
