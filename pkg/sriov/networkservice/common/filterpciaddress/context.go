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

package filterpciaddress

import (
	"context"
)

const (
	PCIAddrListKey contextKeyType = "PCIAddrListKey"
	freeVFInfoKey  contextKeyType = "FreeVFInfo"
)

type contextKeyType string

func WithPCIAddrList(parent context.Context, list []string) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, PCIAddrListKey, list)
}

func PCIAddrList(ctx context.Context) []string {
	if rv, ok := ctx.Value(PCIAddrListKey).([]string); ok {
		return rv
	}
	return nil
}

func WithFreeVFInfo(parent context.Context, freeVFInfo *FreeVirtualFunctionsInfo) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, freeVFInfoKey, freeVFInfo)
}

func FreeVFInfo(ctx context.Context) *FreeVirtualFunctionsInfo {
	if rv, ok := ctx.Value(freeVFInfoKey).(*FreeVirtualFunctionsInfo); ok {
		return rv
	}
	return nil
}
