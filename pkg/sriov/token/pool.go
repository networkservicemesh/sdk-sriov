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

// Package token provides a token pool for SR-IOV resources
package token

import (
	"path"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
)

const (
	free state = iota
	allocated
	inUse
	closed
)

// Pool manages forwarder SR-IOV resource tokens
type Pool struct {
	lock          sync.Mutex
	tokens        map[string]*token   // tokens[id] -> *token
	tokensByNames map[string][]*token // tokensByNames[name] -> []*token
	closedTokens  map[string][]*token // closedTokens[id] -> []*token
	listeners     []func()
}

type state int

func (ts state) String() string {
	if ts < free || closed < ts {
		return "invalid state"
	}
	return []string{
		"free",
		"allocated",
		"inUse",
		"closed",
	}[ts]
}

type token struct {
	id    string
	name  string
	state state
}

// NewPool returns a new Pool
func NewPool(cfg *config.Config) *Pool {
	p := &Pool{
		tokens:        map[string]*token{},
		tokensByNames: map[string][]*token{},
		closedTokens:  map[string][]*token{},
	}

	for _, pf := range cfg.PhysicalFunctions {
		for _, service := range pf.Services {
			for _, capability := range pf.Capabilities {
				name := path.Join(service, capability)
				for i := 0; i < len(pf.VirtualFunctions); i++ {
					tok := &token{
						id:    uuid.New().String(),
						name:  name,
						state: free,
					}
					p.tokens[tok.id] = tok
					p.tokensByNames[tok.name] = append(p.tokensByNames[tok.name], tok)
				}
			}
		}
	}

	return p
}

// AddListener adds a new listener that fires on tokens state change to/from "closed"
func (p *Pool) AddListener(listener func()) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.listeners = append(p.listeners, listener)
}

// Tokens returns a map of tokens by names marked as available/not available
func (p *Pool) Tokens() map[string]map[string]bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	tokens := map[string]map[string]bool{}
	for name, toks := range p.tokensByNames {
		tokens[name] = map[string]bool{}
		for _, tok := range toks {
			tokens[name][tok.id] = tok.state != closed
		}
	}
	return tokens
}

// Find returns a token name selected by the given ID
func (p *Pool) Find(id string) (string, error) {
	tok, err := p.find(id)
	if err != nil {
		return "", err
	}
	return tok.name, nil
}

func (p *Pool) find(id string) (*token, error) {
	if token, ok := p.tokens[id]; ok {
		return token, nil
	}
	return nil, errors.Errorf("token doesn't exist: %s", id)
}

// Allocate marks a token selected by the given ID as "allocated"
func (p *Pool) Allocate(id string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	tok, err := p.find(id)
	if err != nil {
		return err
	}

	if tok.state != free {
		return errors.Errorf("token is not free: %s:%s - %v", tok.name, tok.id, tok.state)
	}
	tok.state = allocated

	return nil
}

// Free marks a token selected by the given ID as "free"
func (p *Pool) Free(id string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	tok, err := p.find(id)
	if err != nil {
		return err
	}

	if tok.state == inUse {
		_ = p.stopUsing(id)
	}
	tok.state = free

	return nil
}

// Use marks a token selected by the given ID as "inUse" and closes 1 token for any of names
func (p *Pool) Use(id string, names []string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	tok, err := p.find(id)
	if err != nil {
		return err
	}

	if tok.state == closed {
		return errors.Errorf("token is closed: %s:%s", tok.name, tok.id)
	}
	tok.state = inUse

	for i := range names {
		if names[i] == tok.name {
			continue
		}

		tokToClose := p.findToClose(names[i])
		if tokToClose == nil {
			continue
		}
		tokToClose.state = closed

		p.closedTokens[tok.id] = append(p.closedTokens[tok.id], tokToClose)
	}

	for _, listener := range p.listeners {
		go listener()
	}

	return nil
}

func (p *Pool) findToClose(name string) *token {
	for _, tok := range p.tokensByNames[name] {
		if tok.state == free {
			return tok
		}
	}
	for _, tok := range p.tokensByNames[name] {
		if tok.state == allocated {
			return tok
		}
	}
	return nil
}

// StopUsing marks an "inUse" token selected by ID as "allocated" and frees all related closed tokens
func (p *Pool) StopUsing(id string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.stopUsing(id)
}

func (p *Pool) stopUsing(id string) error {
	tok, err := p.find(id)
	if err != nil {
		return err
	}

	if tok.state != inUse {
		return errors.Errorf("token is not in use: %s:%s - %v", tok.name, tok.id, tok.state)
	}
	tok.state = allocated

	for _, t := range p.closedTokens[tok.id] {
		t.state = free
	}
	delete(p.closedTokens, tok.id)

	for _, listener := range p.listeners {
		go listener()
	}

	return nil
}
