// Copyright (c) 2023 Doc.ai and/or its affiliates.
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

// Package yamlhelper provides YAML marshaling utils
package yamlhelper

import (
	"os"
	"path"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// UnmarshalFile unmarshal YAML file into the object
func UnmarshalFile(fileName string, o interface{}) error {
	bytes, err := os.ReadFile(path.Clean(fileName))
	if err != nil {
		return errors.Wrapf(err, "error reading file: %v", fileName)
	}

	if err = yaml.Unmarshal(bytes, o); err != nil {
		return errors.Wrapf(err, "error unmarshalling yaml: %s", bytes)
	}

	return nil
}
