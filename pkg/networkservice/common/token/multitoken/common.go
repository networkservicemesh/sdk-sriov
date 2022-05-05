// Copyright (c) 2021-2022 Nordix Foundation.
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

package multitoken

import (
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type tokenElement struct {
	lock                sync.Mutex
	tokens              map[string][]string // tokens[tokenName] -> []tokenIDs
	connectionsByTokens map[string]string   // connectionsByTokens[tokenID] -> connectionID
	tokensByConnections map[string]string   // tokensByConnections[connectionID] -> tokenID
}

type tokenConfig interface {
	assign(tokenName string, conn *networkservice.Connection) (tokenID string)
	get(conn *networkservice.Connection) (tokenID string)
	release(conn *networkservice.Connection)
}

func createTokenElement(allocatableTokens map[string][]string) tokenConfig {
	return &tokenElement{tokens: allocatableTokens,
		connectionsByTokens: map[string]string{},
		tokensByConnections: map[string]string{}}
}

func (c *tokenElement) assign(tokenName string, conn *networkservice.Connection) (tokenID string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var ok bool
	if tokenID, ok = c.tokensByConnections[conn.GetId()]; ok {
		return tokenID
	}

	for _, tokenID = range c.tokens[tokenName] {
		if _, ok := c.connectionsByTokens[tokenID]; !ok {
			c.connectionsByTokens[tokenID] = conn.GetId()
			c.tokensByConnections[conn.GetId()] = tokenID
			break
		} else {
			tokenID = ""
		}
	}
	return
}

func (c *tokenElement) get(conn *networkservice.Connection) (tokenID string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.tokensByConnections[conn.GetId()]
}

func (c *tokenElement) release(conn *networkservice.Connection) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if tokenID, ok := c.tokensByConnections[conn.GetId()]; ok {
		delete(c.connectionsByTokens, tokenID)
		delete(c.tokensByConnections, conn.GetId())
	}
}
