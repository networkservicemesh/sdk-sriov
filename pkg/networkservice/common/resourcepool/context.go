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
	"sync"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/resourcepool"
)

const (
	resourcePoolKey Key = "resourcepool.PCIResourcePool"
)

// Key is a key type for context value
type Key string

// WithResourcePool returns a new context with PCIResourcePool
func WithResourcePool(parent context.Context, resourcePool *resourcepool.ResourcePool, lock *sync.Mutex) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, resourcePoolKey, struct {
		*resourcepool.ResourcePool
		*sync.Mutex
	}{
		resourcePool,
		lock,
	})
}

// GetResourcePool returns PCIResourcePool from context
func GetResourcePool(ctx context.Context) PCIResourcePool {
	if rv, ok := ctx.Value(resourcePoolKey).(PCIResourcePool); ok {
		return rv
	}
	return nil
}
