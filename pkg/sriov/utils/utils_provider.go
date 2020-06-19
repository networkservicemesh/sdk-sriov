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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"

	"github.com/pkg/errors"
)

const (
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

// SriovUtilsProvider provides utility methods for SR-IOV machinery
type SriovUtilsProvider interface {
	// IsDeviceSriovCapable checks if a pci device is SR-IOV capable
	IsDeviceSriovCapable(pciAddr string) bool

	// IsSriovVirtualFunction checks if a pci device has link to a physical function
	IsSriovVirtualFunction(pciAddr string) bool

	// GetConfiguredVirtualFunctionsNumber returns number of virtual functions configured for a given physical function
	GetConfiguredVirtualFunctionsNumber(pfPciAddr string) (int, error)

	// IsSriovConfigured returns true if sriov_numvfs reads > 0 else false
	IsSriovConfigured(pciAddr string) bool

	// GetSriovVirtualFunctionsCapacity returns number of virtual functions that can be created for specified
	// physical function
	GetSriovVirtualFunctionsCapacity(pfPciAddr string) (int, error)

	// IsDeviceExists validates pciAddr given as string and checks if device with it exists
	// PCI addresses in the /sys/bus/pci/devices/ are represented in the extended form of the BDF notation
	// The format is: DDDD:BB:DD.F. That is Domain:Bus:Device.Function.
	// See https://wiki.xen.org/wiki/Bus:Device.Function_(BDF)_Notation for more detailed description
	IsDeviceExists(pciAddr string) (bool, error)

	// GetNetInterfacesNames returns host net interface names as string for a PCI device from its pci address
	GetNetInterfacesNames(pciAddr string) ([]string, error)

	// CreateVirtualFunctions initializes virtual functions for specified physical function
	// if virtual functions are already exist, returns error, even if vfNumber is greater than
	// existing functions number
	CreateVirtualFunctions(pfPciAddr string, vfNumber int) error

	// GetVirtualFunctionsList returns a List containing PCI addr for all VF discovered in a given PF
	GetVirtualFunctionsList(pfPciAddr string) (vfList []string, err error)
}

type sriovUtilsProvider struct {
	sysfsDevicesPath string
}

// NewSriovUtilsProvider returns new SriovUtilsProvider instance
// 			- sysfsDevicesPath - path to directory where sysfs PCI device files are placed. Usually /sys/bus/pci/devices/
func NewSriovUtilsProvider(sysfsDevicesPath string) SriovUtilsProvider {
	return &sriovUtilsProvider{
		sysfsDevicesPath: sysfsDevicesPath,
	}
}

func (u *sriovUtilsProvider) IsDeviceSriovCapable(pciAddr string) bool {
	// sriov_totalvfs file exists -> sriov capable
	return u.isFileExists(filepath.Join(u.sysfsDevicesPath, pciAddr, totalVfFile))
}

func (u *sriovUtilsProvider) IsSriovVirtualFunction(pciAddr string) bool {
	return u.isFileExists(filepath.Join(u.sysfsDevicesPath, pciAddr, physicalFunctionPath))
}

func (u *sriovUtilsProvider) GetConfiguredVirtualFunctionsNumber(pfPciAddr string) (int, error) {
	configuredVfPath := filepath.Join(u.sysfsDevicesPath, pfPciAddr, configuredVfFile)
	vfs, err := ioutil.ReadFile(filepath.Clean(configuredVfPath))
	if err != nil {
		return 0, errors.Errorf("unable to locate sriov_numvfs file for device %s: %v", pfPciAddr, err)
	}
	configuredVFs := bytes.TrimSpace(vfs)
	numConfiguredVFs, err := strconv.Atoi(string(configuredVFs))
	if err != nil {
		return 0, errors.Errorf("unable to convert string to int from sriov_numvfs file for device %s: %v", pfPciAddr, err)
	}
	return numConfiguredVFs, nil
}

func (u *sriovUtilsProvider) IsSriovConfigured(pciAddr string) bool {
	numVfs, err := u.GetConfiguredVirtualFunctionsNumber(pciAddr)
	if err != nil {
		return false
	}
	return numVfs > 0
}

func (u *sriovUtilsProvider) GetSriovVirtualFunctionsCapacity(pfPciAddr string) (int, error) {
	totalVfFilePath := filepath.Join(u.sysfsDevicesPath, pfPciAddr, totalVfFile)
	vfs, err := ioutil.ReadFile(filepath.Clean(totalVfFilePath))
	if err != nil {
		return 0, errors.Errorf("unable to locate sriov_totalvfs file for device %s: %v", pfPciAddr, err)
	}
	totalVfs := bytes.TrimSpace(vfs)
	numTotalVfs, err := strconv.Atoi(string(totalVfs))
	if err != nil {
		return 0, errors.Errorf("unable to convert string to int from sriov_totalvfs file for device %s: %v", pfPciAddr, err)
	}
	return numTotalVfs, nil
}

func (u *sriovUtilsProvider) IsDeviceExists(pciAddr string) (bool, error) {
	// init sysfs pci address regex
	initRegexpOnce.Do(func() {
		validLongPCIAddr = regexp.MustCompile(`^0{4}:[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)
		validShortPCIAddr = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)
	})

	// Check system pci address
	if validLongPCIAddr.MatchString(pciAddr) {
		devicePath := filepath.Join(u.sysfsDevicesPath, pciAddr)
		return u.isFileOrLinkExists(devicePath), nil
	} else if validShortPCIAddr.MatchString(pciAddr) {
		pciAddr = bdfDomain + pciAddr // convert short form to sysfs' long form representation
		devicePath := filepath.Join(u.sysfsDevicesPath, pciAddr)
		return u.isFileOrLinkExists(devicePath), nil
	}
	return false, errors.Errorf("invalid pci address provided: %s", pciAddr)
}

func (u *sriovUtilsProvider) GetNetInterfacesNames(pciAddr string) ([]string, error) {
	var names []string
	netDir := filepath.Join(u.sysfsDevicesPath, pciAddr, netInterfacesPath)
	if _, err := os.Lstat(netDir); err != nil {
		return nil, errors.Errorf("no net directory under pci device %s: %v", pciAddr, err)
	}

	fInfos, err := ioutil.ReadDir(netDir)
	if err != nil {
		return nil, errors.Errorf("failed to read net directory %s: %v", netDir, err)
	}

	names = make([]string, 0)
	for _, f := range fInfos {
		names = append(names, f.Name())
	}
	return names, nil
}

func (u *sriovUtilsProvider) CreateVirtualFunctions(pfPciAddr string, vfNumber int) error {
	if vfNumber < 1 {
		return errors.Errorf("invalid number of virtual functions specified: %d. Must be > 0", vfNumber)
	}
	if u.IsSriovConfigured(pfPciAddr) {
		return errors.Errorf("virtual functions are already exist for device: %s", pfPciAddr)
	}
	configuredVfPath := filepath.Join(u.sysfsDevicesPath, pfPciAddr, configuredVfFile)
	err := ioutil.WriteFile(configuredVfPath, []byte(strconv.FormatInt(int64(vfNumber), 10)), 0600)
	if err != nil {
		return errors.Errorf("failed to write virtual functions number for device %s: %v", pfPciAddr, err)
	}
	return nil
}

func (u *sriovUtilsProvider) GetVirtualFunctionsList(pfPciAddr string) (vfList []string, err error) {
	vfList = make([]string, 0)
	pfDir := filepath.Join(u.sysfsDevicesPath, pfPciAddr)
	_, err = os.Lstat(pfDir)
	if err != nil {
		err = errors.Errorf("could not get PF directory information for device %s: %v", pfPciAddr, err)
		return
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))
	if err != nil {
		err = errors.Errorf("error reading VF directories %v", err)
		return
	}

	// Read all VF directory and get add VF PCI addr to the vfList
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				vfLink := filepath.Base(linkName)
				vfList = append(vfList, vfLink)
			}
		}
	}
	return
}

func (u *sriovUtilsProvider) isFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func (u *sriovUtilsProvider) isFileOrLinkExists(filePath string) bool {
	_, err := os.Lstat(filePath)
	return err == nil
}
