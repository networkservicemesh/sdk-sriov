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

// Package utils contains useful helper methods for network machinery
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
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	sysBusPci        = "/sys/bus/pci/devices"
	totalVfFile      = "sriov_totalvfs"
	configuredVfFile = "sriov_numvfs"
	bdfDomain        = "0000:"
)

var (
	initRegexpOnce    sync.Once
	validLongPCIAddr  *regexp.Regexp
	validShortPCIAddr *regexp.Regexp
)

// IsDeviceSriovCapable checks if a pci device is SR-IOV capable
func IsDeviceSriovCapable(pciAddr string) bool {
	// sriov_totalvfs file exists -> sriov capable
	return isFileExists(filepath.Join(sysBusPci, pciAddr, totalVfFile))
}

// IsSriovVirtualFunction checks if a pci device has link to a physical function
func IsSriovVirtualFunction(pciAddr string) bool {
	return isFileExists(filepath.Join(sysBusPci, pciAddr, "physfn"))
}

// GetConfiguredVirtualFunctionsNumber returns number of virtual functions configured for a given physical function
func GetConfiguredVirtualFunctionsNumber(pfPciAddr string) (int, error) {
	configuredVfPath := filepath.Join(sysBusPci, pfPciAddr, configuredVfFile)
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

// IsSriovConfigured returns true if sriov_numvfs reads > 0 else false
func IsSriovConfigured(pciAddr string) bool {
	numVfs, err := GetConfiguredVirtualFunctionsNumber(pciAddr)
	if err != nil {
		return false
	}
	return numVfs > 0
}

// GetSriovVirtualFunctionsCapacity returns number of virtual functions that can be created for specified
// physical function
func GetSriovVirtualFunctionsCapacity(pfPciAddr string) (int, error) {
	totalVfFilePath := filepath.Join(sysBusPci, pfPciAddr, totalVfFile)
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

// IsDeviceExists validates pciAddr given as string and checks if device with it exists
// PCI addresses in the /sys/bus/pci/devices/ are represented in the extended form of the BDF notation
// The format is: DDDD:BB:DD.F. That is Domain:Bus:Device.Function.
// See https://wiki.xen.org/wiki/Bus:Device.Function_(BDF)_Notation for more detailed description
func IsDeviceExists(pciAddr string) error {
	// Check system pci address

	// init sysfs pci address regex
	initRegexpOnce.Do(func() {
		validLongPCIAddr = regexp.MustCompile(`^0{4}:[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)
		validShortPCIAddr = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)
	})

	if validLongPCIAddr.MatchString(pciAddr) {
		return isDeviceExists(pciAddr)
	} else if validShortPCIAddr.MatchString(pciAddr) {
		pciAddr = bdfDomain + pciAddr // convert short form to sysfs' long form representation
		return isDeviceExists(pciAddr)
	}
	return errors.Errorf("invalid pci address %s", pciAddr)
}

// GetNetInterfacesNames returns host net interface names as string for a PCI device from its pci address
func GetNetInterfacesNames(pciAddr string) ([]string, error) {
	var names []string
	netDir := filepath.Join(sysBusPci, pciAddr, "net")
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

// IsDefaultRoute returns true if PCI network device is default route interface
func IsDefaultRoute(pciAddr string) (bool, error) {
	// Get net interface name
	ifNames, err := GetNetInterfacesNames(pciAddr)
	if err != nil {
		return false, errors.Errorf("error trying get net device name for device %s", pciAddr)
	}

	if len(ifNames) > 0 { // there's at least one interface name found
		for _, ifName := range ifNames {
			link, err := netlink.LinkByName(ifName)
			if err != nil {
				logrus.Errorf("expected to get valid host interface with name %s: %q", ifName, err)
				continue
			}

			routes, err := netlink.RouteList(link, netlink.FAMILY_V4) // IPv6 routes: all interface has at least one link local route entry
			if err != nil {
				logrus.Errorf("expected to get valid routes for interface with name %s: %q", ifName, err)
				continue
			}

			for idx := range routes {
				if routes[idx].Dst == nil {
					logrus.Infof("excluding interface %s: default route found: %+v", ifName, routes[idx])
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// CreateVirtualFunctions initializes VFs for specified PF given number of VFs
func CreateVirtualFunctions(pfPciAddr string, vfNumber int) error {
	configuredVfPath := filepath.Join(sysBusPci, pfPciAddr, configuredVfFile)
	err := ioutil.WriteFile(configuredVfPath, []byte(strconv.FormatInt(int64(vfNumber), 10)), 0600)
	if err != nil {
		return err
	}
	return nil
}

// GetVirtualFunctionsList returns a List containing PCI addr for all VF discovered in a given PF
func GetVirtualFunctionsList(pfPciAddr string) (vfList []string, err error) {
	vfList = make([]string, 0)
	pfDir := filepath.Join(sysBusPci, pfPciAddr)
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

func isFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func isDeviceExists(pciAddr string) error {
	devPath := filepath.Join(sysBusPci, pciAddr)
	_, err := os.Lstat(devPath)
	if err != nil {
		return errors.Errorf("unable to read device directory %s: %v", devPath, err)
	}
	return nil
}
