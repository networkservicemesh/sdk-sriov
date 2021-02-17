// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package vfio

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var devicesCgroup = regexp.MustCompile("^[1-9][0-9]*?:devices:(.*)$")

func cgroupDirPath() (string, error) {
	cgroupInfo, err := os.OpenFile("/proc/self/cgroup", os.O_RDONLY, 0)
	if err != nil {
		return "", errors.Wrap(err, "error opening cgroup info file")
	}
	for scanner := bufio.NewScanner(cgroupInfo); scanner.Scan(); {
		line := scanner.Text()
		if devicesCgroup.MatchString(line) {
			return podCgroupDirPath(devicesCgroup.FindStringSubmatch(line)[1]), nil
		}
	}
	return "", errors.New("can't find out cgroup directory")
}

func podCgroupDirPath(containerCgroupDirPath string) string {
	split := strings.Split(containerCgroupDirPath, string(filepath.Separator))
	split[len(split)-1] = "*" // any container match
	return filepath.Join(split...)
}
