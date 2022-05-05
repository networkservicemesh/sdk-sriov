// Copyright (c) 2021-2022 Nordix Foundation.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

//go:build linux
// +build linux

package token

import (
	"os"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token/multitoken"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token/sharedtoken"
	"github.com/networkservicemesh/sdk-sriov/pkg/tools/tokens"
)

// NewServer returns a new token server chain element for the given tokenKey
func NewServer(tokenKey string) networkservice.NetworkServiceServer {
	sriovTokens := tokens.FromEnv(os.Environ())[tokenKey]
	if len(sriovTokens) == 1 {
		return sharedtoken.NewServer(sriovTokens[0])
	}
	return multitoken.NewServer(tokenKey)
}
