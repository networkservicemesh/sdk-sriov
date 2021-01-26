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

//+build !windows

package vfio

import (
	"context"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

type vfioClient struct {
	vfioDir   string
	cgroupDir string
}

const (
	mkdirPerm = 0750
	mknodPerm = 0666
)

// NewClient returns a new VFIO client chain element
func NewClient(vfioDir, cgroupDir string) networkservice.NetworkServiceClient {
	return &vfioClient{
		vfioDir:   vfioDir,
		cgroupDir: cgroupDir,
	}
}

func (c *vfioClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logEntry := logger.Log(ctx).WithField("vfioClient", "Request")

	request.MechanismPreferences = append(request.MechanismPreferences, vfio.New(c.cgroupDir))

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if mech := vfio.ToMechanism(conn.GetMechanism()); mech != nil {
		if err := os.Mkdir(c.vfioDir, mkdirPerm); err != nil && !os.IsExist(err) {
			logEntry.Error("failed to create vfio directory")
			return nil, err
		}

		if err := unix.Mknod(
			filepath.Join(c.vfioDir, vfioDevice),
			unix.S_IFCHR|mknodPerm,
			int(unix.Mkdev(mech.GetVfioMajor(), mech.GetVfioMinor())),
		); err != nil && !os.IsExist(err) {
			logEntry.Errorf("failed to mknod device: %v", vfioDevice)
			return nil, err
		}

		igid := mech.GetParameters()[vfio.IommuGroupKey]
		if err := unix.Mknod(
			filepath.Join(c.vfioDir, igid),
			unix.S_IFCHR|mknodPerm,
			int(unix.Mkdev(mech.GetDeviceMajor(), mech.GetDeviceMinor())),
		); err != nil && !os.IsExist(err) {
			logEntry.Errorf("failed to mknod device: %v", vfioDevice)
			return nil, err
		}
	}

	return conn, nil
}

func (c *vfioClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
