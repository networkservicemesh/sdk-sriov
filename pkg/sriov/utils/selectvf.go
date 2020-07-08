package utils

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
)

// SelectVirtualFunction marks one of the free virtual functions for specified physical function as in-use and returns it
func SelectVirtualFunction(pfPCIAddr string, resourcePool *sriov.NetResourcePool) (selectedVf *sriov.VirtualFunction, err error) {
	resourcePool.Lock()
	defer resourcePool.Unlock()

	for _, netResource := range resourcePool.Resources {
		pf := netResource.PhysicalFunction
		if pf.PCIAddress != pfPCIAddr {
			continue
		}

		// select the first free virtual function
		for vf, state := range pf.VirtualFunctions {
			if state == sriov.FreeVirtualFunction {
				selectedVf = vf
				break
			}
		}
		if selectedVf == nil {
			return nil, errors.Errorf("no free virtual function found for device %s", pfPCIAddr)
		}

		// mark it as in use
		err = pf.SetVirtualFunctionState(selectedVf, sriov.UsedVirtualFunction)
		if err != nil {
			return nil, err
		}

		return selectedVf, nil
	}

	return nil, errors.Errorf("no physical function with PCI address %s found", pfPCIAddr)
}

// FreeVirtualFunction marks given virtual function as free
func FreeVirtualFunction(pfPCIAddr, vfNetIfaceName string, resourcePool *sriov.NetResourcePool) error {
	resourcePool.Lock()
	defer resourcePool.Unlock()

	for _, netResource := range resourcePool.Resources {
		pf := netResource.PhysicalFunction
		if pf.PCIAddress != pfPCIAddr {
			continue
		}

		for vf := range pf.VirtualFunctions {
			if vf.NetInterfaceName == vfNetIfaceName {
				return pf.SetVirtualFunctionState(vf, sriov.FreeVirtualFunction)
			}
		}
		return errors.Errorf("no virtual function with net interface name %s found", vfNetIfaceName)
	}
	return errors.Errorf("no physical function with PCI address %s found", pfPCIAddr)
}
