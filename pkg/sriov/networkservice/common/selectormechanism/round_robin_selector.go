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

// Package selectormechanism provides a selection networkservice mechanism by round robins algorithm
package selectormechanism

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"go.uber.org/atomic"
)

type roundRobinSelector struct {
	index *atomic.Int32
}

func newRoundRobinSelector() *roundRobinSelector {
	return &roundRobinSelector{index: atomic.NewInt32(0)}
}

func (rr *roundRobinSelector) selectMechanism(mechanisms []*networkservice.Mechanism) *networkservice.Mechanism {
	if rr == nil || len(mechanisms) == 0 {
		return nil
	}

	idx := rr.index.Load() % int32(len(mechanisms))
	mech := mechanisms[idx]
	if mech == nil {
		return nil
	}
	rr.index.Inc()

	return mech
}
