// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

//go:build !windows
// +build !windows

package cgroup

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"

	"golang.org/x/sys/unix"
)

const (
	createPerm = 0o666
)

func inputFileAPI(ctx context.Context, filePath string, consumer func(string)) error {
	_ = os.Remove(filePath)
	err := unix.Mkfifo(filePath, createPerm)
	if err != nil {
		return err
	}

	fd, err := unix.Open(filePath, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), filePath)

	go func() {
		defer func() { _ = file.Close() }()
		for fileCh := readFile(ctx, file); ; {
			select {
			case <-ctx.Done():
				return
			case input := <-fileCh:
				consumer(input)
			}
		}
	}()

	return nil
}

func readFile(ctx context.Context, file *os.File) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for scanner := bufio.NewScanner(file); scanner.Scan(); {
			select {
			case <-ctx.Done():
				return
			case ch <- scanner.Text():
			}
		}
	}()
	return ch
}

type supplierFunc func(s string) error

func outputFileAPI(filePath string) (supplier supplierFunc) {
	supplier = func(data string) error {
		return ioutil.WriteFile(filePath, []byte(data), createPerm)
	}
	return supplier
}
