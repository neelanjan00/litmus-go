package lib

import (
	"os"

	clients "github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/cloud/vmware"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/vmware/vmware-host-reboot/types"
	"github.com/pkg/errors"
)

// PrepareHostReboot executes the chaos prepration, injection, and revert steps
func PrepareHostReboot(experimentsDetails *experimentTypes.ExperimentDetails, hostId, cookie string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	VMDisks := make(map[string][]string)

	// Waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	// Get the ids of the VMs that are attached to the host and are powered-on and powered-off or suspended
	log.Infof("[Info]: Fetching the VMs attached to the host")
	poweredOnVMList, poweredOffOrSuspendedVMList, err := vmware.GetVMDetails(experimentsDetails.VcenterServer, hostId, cookie)
	if err != nil {
		return errors.Errorf("failed to fetch the VM details: %s", err.Error())
	}

	// Get the disks attached to the VMs
	log.Infof("[Info]: Fetching the disks attached to the VMs that are attached to the host")
	for _, vmId := range append(poweredOnVMList, poweredOffOrSuspendedVMList...) {

		diskIdList, err := vmware.GetDisks(experimentsDetails.VcenterServer, vmId, cookie)
		if err != nil {
			return errors.Errorf("failed to fetch the disk details for %s vm: %s", vmId, err.Error())
		}

		VMDisks[vmId] = diskIdList
	}

	// Set the ENVs for govc
	os.Setenv("GOVC_URL", experimentsDetails.VcenterServer)
	os.Setenv("GOVC_USERNAME", experimentsDetails.VcenterUser)
	os.Setenv("GOVC_PASSWORD", experimentsDetails.VcenterPass)
	os.Setenv("GOVC_INSECURE", "true")

	// Reboot the host
	log.Info("[Chaos]: Rebooting the ESX host")
	if err := vmware.RebootHost(experimentsDetails.HostName, experimentsDetails.HostDatacenter); err != nil {
		return errors.Errorf("failed to start host reboot: %s", err.Error())
	}

	// Wait for the host to completely reboot
	log.Info("[Wait]: Wait for the host to completely reboot")
	if err := vmware.WaitForHostToDisconnect(experimentsDetails.Timeout, experimentsDetails.Delay, experimentsDetails.VcenterServer, experimentsDetails.HostName, cookie); err != nil {
		return errors.Errorf("host failed to successfully reboot: %s", err.Error())
	}

	if err := vmware.WaitForHostToConnect(experimentsDetails.Timeout, experimentsDetails.Delay, experimentsDetails.VcenterServer, experimentsDetails.HostName, cookie); err != nil {
		return errors.Errorf("host failed to successfully reboot: %s", err.Error())
	}

	// Power-on the VMs that were powered-on prior to the host reboot
	for _, vmId := range poweredOnVMList {

		log.Infof("[Info]: Starting the %s VM", vmId)
		if err := vmware.StartVM(experimentsDetails.VcenterServer, vmId, cookie); err != nil {
			return errors.Errorf("failed to start the %s vm, %s", vmId, err.Error())
		}

		log.Infof("[Wait]: Wait for the %s VM to start", vmId)
		if err := vmware.WaitForVMStart(experimentsDetails.Timeout, experimentsDetails.Delay, experimentsDetails.VcenterServer, vmId, cookie); err != nil {
			return errors.Errorf("%s vm failed to successfully start, %s", vmId, err.Error())
		}
	}

	// Check if the disks are still attached to their respective VMs
	for vmId, diskList := range VMDisks {

		log.Infof("[Info]: Checking the attachment status of the disks of the %s vm", vmId)
		for _, diskId := range diskList {

			diskState, err := vmware.GetDiskState(experimentsDetails.VcenterServer, vmId, diskId, cookie)
			if err != nil {
				return errors.Errorf("failed to get the %s disk state for %s vm", diskId, vmId)
			}

			if diskState != "attached" {
				return errors.Errorf("disk state is %s for disk %s of vm %s", diskState, diskId, vmId)
			}
		}
	}

	// Waiting for the ramp time after chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	return nil
}
