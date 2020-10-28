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

package token

import (
	"encoding/json"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/storage"
)

type tokenStorage struct {
	storage storage.Storage
}

type storedToken struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *tokenStorage) store(tokens map[string]*token) {
	state := map[string]string{}
	for id, tok := range tokens {
		state[id] = marshallToken(tok)
	}
	s.storage.Store(state)
}

func marshallToken(tok *token) string {
	data, _ := json.Marshal(&storedToken{
		ID:   tok.id,
		Name: tok.name,
	})
	return string(data)
}

func (s *tokenStorage) load() map[string]*token {
	tokens := map[string]*token{}
	state := s.storage.Load()
	for id, s := range state {
		tokens[id] = unmarshallToken(s)
	}
	return tokens
}

func unmarshallToken(s string) (tok *token) {
	storedTok := storedToken{}
	_ = json.Unmarshal([]byte(s), &storedTok)

	return &token{
		id:    storedTok.ID,
		name:  storedTok.Name,
		state: free,
	}
}
