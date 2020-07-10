package utils

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"unicode"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netns"
)

// KernelProvider provides utility methods for Kernel mechanism machinery
type KernelProvider interface {
	// MoveInterfaceToAnotherNamespace moves specified network interface from fromNetNS to toNetNS network namespace
	MoveInterfaceToAnotherNamespace(ifaceName string, fromNetNS, toNetNS netns.NsHandle) error

	// GetNSHandleFromInode returns namespace handler from inode
	GetNSHandleFromInode(inode string) (netns.NsHandle, error)
}

type kernelProvider struct {
}

// NewKernelProvider returns new KernelProvider instance
func NewKernelProvider() KernelProvider {
	return &kernelProvider{}
}

func (k *kernelProvider) MoveInterfaceToAnotherNamespace(ifaceName string, fromNetNS, toNetNS netns.NsHandle) error {
	link, err := kernel.FindHostDevice("", ifaceName, fromNetNS)
	if err != nil {
		return err
	}

	err = link.MoveToNetns(toNetNS)
	if err != nil {
		return errors.Wrapf(err, "Failed to move interface %s to another namespace", ifaceName)
	}

	return nil
}

func (k *kernelProvider) GetNSHandleFromInode(inode string) (netns.NsHandle, error) {
	/* Parse the string to an integer */
	inodeNum, err := strconv.ParseUint(inode, 10, 64)
	if err != nil {
		return -1, errors.Errorf("failed parsing inode, must be an unsigned int, instead was: %s", inode)
	}
	/* Get filepath from inode */
	nsPath, err := resolvePodNSByInode(inodeNum)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to find file in /proc/*/ns/net with inode %d", inodeNum)
	}
	/* Get namespace handler from nsPath */
	return netns.GetFromPath(nsPath)
}

// resolvePodNSByInode Traverse /proc/<pid>/<suffix> files,
// compare their inodes with inode parameter and returns file if inode matches
func resolvePodNSByInode(inode uint64) (string, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return "", errors.Wrap(err, "can't read /proc directory")
	}

	for _, f := range files {
		name := f.Name()
		if isDigits(name) {
			filename := path.Join("/proc", name, "/ns/net")
			tryInode, err := getInode(filename)
			if err != nil {
				// Just report into log, do not exit
				logrus.Errorf("Can't find %s Error: %v", filename, err)
				continue
			}
			if tryInode == inode {
				if cmdline, err := getCmdline(name); err == nil && strings.Contains(cmdline, "pause") {
					return filename, nil
				}
			}
		}
	}

	return "", errors.New("not found")
}

func isDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func getInode(file string) (uint64, error) {
	fileinfo, err := os.Stat(file)
	if err != nil {
		return 0, errors.Wrap(err, "error stat file")
	}
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("not a stat_t")
	}
	return stat.Ino, nil
}

func getCmdline(pid string) (string, error) {
	data, err := ioutil.ReadFile(path.Clean(path.Join("/proc/", pid, "cmdline")))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
