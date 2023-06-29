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

package pcifunction

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func isFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readUintFromFile(path string) (uint, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return 0, errors.Wrapf(err, "unable to locate file: %v", path)
	}

	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, errors.Wrapf(err, "unable to convert string to int: %v", string(data))
	}

	return uint(value), nil
}

func evalSymlinkAndGetBaseName(path string) (string, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return "", errors.Wrapf(err, "error getting info about specified file: %s", path)
	}
	if fileInfo.Mode()&os.ModeSymlink == 0 {
		return "", errors.Errorf("specified file is not a symbolic link: %s", path)
	}

	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.Wrapf(err, "error evaluating symbolic link: %s", path)
	}

	realPathBase := filepath.Base(realPath)

	return realPathBase, nil
}
