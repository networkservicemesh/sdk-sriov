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

// Package utils provides utils for SR-IOV machinery
package utils

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	// SysfsDevicesPath is path to the devices info in Linux sysfs pseudo file system
	SysfsDevicesPath     = "/sys/bus/pci/devices/"
	totalVfFile          = "sriov_totalvfs"
	configuredVfFile     = "sriov_numvfs"
	physicalFunctionPath = "physfn"
	netInterfacesPath    = "net"
	bdfDomain            = "0000:"
)

var (
	initRegexpOnce    sync.Once
	validLongPCIAddr  *regexp.Regexp
	validShortPCIAddr *regexp.Regexp
)

// SriovProvider provides utility methods for SR-IOV machinery
type SriovProvider interface {
	// IsDeviceSriovCapable checks if a pci device is SR-IOV capable
	IsDeviceSriovCapable(ctx context.Context, pciAddr string) bool

	// IsSriovVirtualFunction checks if a pci device has link to a physical function
	IsSriovVirtualFunction(ctx context.Context, pciAddr string) bool

	// GetConfiguredVirtualFunctionsNumber returns number of virtual functions configured for a given physical function
	GetConfiguredVirtualFunctionsNumber(ctx context.Context, pfPciAddr string) (int, error)

	// IsSriovConfigured returns true if sriov_numvfs reads > 0 else false
	IsSriovConfigured(ctx context.Context, pciAddr string) bool

	// GetSriovVirtualFunctionsCapacity returns number of virtual functions that can be created for specified
	// physical function
	GetSriovVirtualFunctionsCapacity(ctx context.Context, pfPciAddr string) (int, error)

	// IsDeviceExists validates pciAddr given as string and checks if device with it exists
	// PCI addresses in the /sys/bus/pci/devices/ are represented in the extended form of the BDF notation
	// The format is: DDDD:BB:DD.F. That is Domain:Bus:Device.Function.
	// See https://wiki.xen.org/wiki/Bus:Device.Function_(BDF)_Notation for more detailed description
	IsDeviceExists(ctx context.Context, pciAddr string) (bool, error)

	// GetNetInterfacesNames returns host net interface names as string for a PCI device from its pci address
	GetNetInterfacesNames(ctx context.Context, pciAddr string) ([]string, error)

	// CreateVirtualFunctions initializes virtual functions for specified physical function
	// if virtual functions are already exist, returns error, even if vfNumber is greater than
	// existing functions number
	CreateVirtualFunctions(ctx context.Context, pfPciAddr string, vfNumber int) error

	// GetVirtualFunctionsList returns a List containing PCI addr for all VF discovered in a given PF
	GetVirtualFunctionsList(ctx context.Context, pfPciAddr string) (vfList []string, err error)
}

type sriovProvider struct {
	sysfsDevicesPath string
}

// NewSriovProvider returns new SriovProvider instance
// 			- sysfsDevicesPath - path to directory where sysfs PCI device files are placed. Usually /sys/bus/pci/devices/
func NewSriovProvider(sysfsDevicesPath string) SriovProvider {
	return &sriovProvider{
		sysfsDevicesPath: sysfsDevicesPath,
	}
}

func (u *sriovProvider) IsDeviceSriovCapable(ctx context.Context, pciAddr string) bool {
	// sriov_totalvfs file exists -> sriov capable
	if u.isFileExists(filepath.Join(u.sysfsDevicesPath, pciAddr, totalVfFile)) {
		log.Entry(ctx).Infof("Device %s is SR-IOV capable", pciAddr)
		return true
	}
	log.Entry(ctx).Infof("Device %s is not SR-IOV capable", pciAddr)
	return false
}

func (u *sriovProvider) IsSriovVirtualFunction(ctx context.Context, pciAddr string) bool {
	if u.isFileExists(filepath.Join(u.sysfsDevicesPath, pciAddr, physicalFunctionPath)) {
		log.Entry(ctx).Infof("Device %s is SR-IOV virtual function", pciAddr)
		return true
	}
	log.Entry(ctx).Infof("Device %s is not SR-IOV virtual function", pciAddr)
	return false
}

func (u *sriovProvider) GetConfiguredVirtualFunctionsNumber(ctx context.Context, pfPciAddr string) (int, error) {
	configuredVfPath := filepath.Join(u.sysfsDevicesPath, pfPciAddr, configuredVfFile)
	vfs, err := ioutil.ReadFile(filepath.Clean(configuredVfPath))
	if err != nil {
		return 0, errors.Wrapf(err, "unable to locate sriov_numvfs file for device %s", pfPciAddr)
	}
	configuredVFs := bytes.TrimSpace(vfs)
	numConfiguredVFs, err := strconv.Atoi(string(configuredVFs))
	if err != nil {
		return 0, errors.Wrapf(err, "unable to convert string to int from sriov_numvfs file for device %s", pfPciAddr)
	}
	log.Entry(ctx).Infof("Found %d configured virtual functions for device %s", numConfiguredVFs, pfPciAddr)
	return numConfiguredVFs, nil
}

func (u *sriovProvider) IsSriovConfigured(ctx context.Context, pciAddr string) bool {
	numVfs, err := u.GetConfiguredVirtualFunctionsNumber(ctx, pciAddr)
	if err != nil {
		return false
	}
	return numVfs > 0
}

func (u *sriovProvider) GetSriovVirtualFunctionsCapacity(ctx context.Context, pfPciAddr string) (int, error) {
	totalVfFilePath := filepath.Join(u.sysfsDevicesPath, pfPciAddr, totalVfFile)
	vfs, err := ioutil.ReadFile(filepath.Clean(totalVfFilePath))
	if err != nil {
		return 0, errors.Wrapf(err, "unable to locate sriov_totalvfs file for device %s", pfPciAddr)
	}
	totalVfs := strings.TrimSpace(string(vfs))
	numTotalVfs, err := strconv.Atoi(totalVfs)
	if err != nil {
		return 0, errors.Wrapf(err, "unable to convert string to int from sriov_totalvfs file for device %s", pfPciAddr)
	}
	log.Entry(ctx).Infof("Maximum number of virtual functions for device %s is: %d", pfPciAddr, numTotalVfs)
	return numTotalVfs, nil
}

func (u *sriovProvider) IsDeviceExists(ctx context.Context, pciAddr string) (bool, error) {
	// init sysfs pci address regex
	initRegexpOnce.Do(func() {
		validLongPCIAddr = regexp.MustCompile(`^0{4}:[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)
		validShortPCIAddr = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)
	})
	// Check system pci address
	if validShortPCIAddr.MatchString(pciAddr) {
		pciAddr = bdfDomain + pciAddr // convert short form to sysfs long form representation
	}
	if validLongPCIAddr.MatchString(pciAddr) {
		devicePath := filepath.Join(u.sysfsDevicesPath, pciAddr)
		if u.isFileOrLinkExists(devicePath) {
			log.Entry(ctx).Infof("Found device %s", pciAddr)
			return true, nil
		}
		log.Entry(ctx).Infof("Could not find device %s", pciAddr)
		return false, nil
	}
	return false, errors.Errorf("invalid pci address provided: %s", pciAddr)
}

func (u *sriovProvider) GetNetInterfacesNames(ctx context.Context, pciAddr string) (names []string, err error) {
	netDir := filepath.Join(u.sysfsDevicesPath, pciAddr, netInterfacesPath)
	if !u.isFileOrLinkExists(netDir) {
		return nil, errors.Errorf("no net directory for pci device %s", pciAddr)
	}

	fInfos, err := ioutil.ReadDir(netDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read net directory %s", netDir)
	}

	for _, f := range fInfos {
		names = append(names, f.Name())
		log.Entry(ctx).Infof("Found net interface \"%s\" for device %s", f.Name(), pciAddr)
	}
	return names, nil
}

func (u *sriovProvider) CreateVirtualFunctions(ctx context.Context, pfPciAddr string, vfNumber int) error {
	if vfNumber < 1 {
		return errors.Errorf("invalid number of virtual functions specified: %d. Must be > 0", vfNumber)
	}
	if u.IsSriovConfigured(ctx, pfPciAddr) {
		return errors.Errorf("virtual functions are already exist for device: %s", pfPciAddr)
	}
	configuredVfPath := filepath.Join(u.sysfsDevicesPath, pfPciAddr, configuredVfFile)
	err := ioutil.WriteFile(configuredVfPath, []byte(strconv.FormatInt(int64(vfNumber), 10)), 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to write virtual functions number for device %s", pfPciAddr)
	}
	log.Entry(ctx).Infof("Created %d virtual functions for device %s", vfNumber, pfPciAddr)
	return nil
}

func (u *sriovProvider) GetVirtualFunctionsList(ctx context.Context, pfPciAddr string) (vfList []string, err error) {
	pfDir := filepath.Join(u.sysfsDevicesPath, pfPciAddr)
	if !u.isFileOrLinkExists(pfDir) {
		err = errors.Errorf("could not get physical function directory information for device %s", pfPciAddr)
		return
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))
	if err != nil {
		err = errors.Wrapf(err, "error reading virtual function directories: %s", vfDirs)
		return
	}

	// Read all virtual function directories and get add virtual function PCI address to the vfList
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				vfPciAddr := filepath.Base(linkName)
				vfList = append(vfList, vfPciAddr)
				log.Entry(ctx).Infof("Found virtual function %s for device %s", vfPciAddr, pfPciAddr)
			}
		}
	}
	return
}

func (u *sriovProvider) isFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func (u *sriovProvider) isFileOrLinkExists(filePath string) bool {
	_, err := os.Lstat(filePath)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}
