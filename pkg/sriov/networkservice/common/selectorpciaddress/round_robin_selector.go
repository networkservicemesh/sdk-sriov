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

// Package selectorpciaddress provides a selection networkservice mechanism by round robins algorithm
package selectorpciaddress

import (
	"sync/atomic"
)

// RoundRobinSelector contains index for selecting
type RoundRobinSelector struct {
	index int32
}

// NewRoundRobinSelector create new selector
func NewRoundRobinSelector() *RoundRobinSelector {
	return &RoundRobinSelector{index: 0}
}

// SelectStringItem select item drom list
func (rr *RoundRobinSelector) SelectStringItem(items []string) string {
	if rr == nil || len(items) == 0 {
		return ""
	}

	idx := atomic.LoadInt32(&rr.index) % int32(len(items))
	item := items[idx]
	if item == "" {
		return ""
	}
	atomic.AddInt32(&rr.index, 1)

	return item
}
