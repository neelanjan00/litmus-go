package lib

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/cloud/vmware"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/vmware/vmware-disk-loss/types"
	"github.com/pkg/errors"
)

var (
	err           error
	inject, abort chan os.Signal
)

//PrepareDiskLoss contains the prepration and injection steps for the experiment
func PrepareDiskLoss(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails, cookie string) error {

	var diskPathList []string

	// inject channel is used to transmit signal notifications.
	inject = make(chan os.Signal, 1)
	// Catch and relay certain signal(s) to inject channel.
	signal.Notify(inject, os.Interrupt, syscall.SIGTERM)

	// abort channel is used to transmit signal notifications.
	abort = make(chan os.Signal, 1)
	// Catch and relay certain signal(s) to abort channel.
	signal.Notify(abort, os.Interrupt, syscall.SIGTERM)

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	//get the disk id list
	diskIdList := strings.Split(experimentsDetails.DiskIds, ",")
	if len(diskIdList) == 0 {
		return errors.Errorf("no disk ids found to detach")
	}

	//get the vm id list
	appVMMoidList := strings.Split(experimentsDetails.AppVMMoids, ",")
	if len(appVMMoidList) == 0 {
		return errors.Errorf("no vm ids found for corresponding disks")
	}

	if len(diskIdList) != len(appVMMoidList) {
		return errors.Errorf("unequal number of disk ids and vm ids found")
	}

	//get the disk paths for the given disk ids
	for i := range diskIdList {

		diskPath, err := vmware.GetDiskPath(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie)
		if err != nil {
			return errors.Errorf("failed to get the disk path, err: %v", err.Error())
		}

		diskPathList = append(diskPathList, diskPath)
	}

	select {
	case <-inject:
		// stopping the chaos execution, if abort signal recieved
		os.Exit(0)
	default:

		// watching for the abort signal and revert the chaos
		go AbortWatcher(experimentsDetails, diskIdList, diskPathList, appVMMoidList, cookie, abort, chaosDetails)

		switch strings.ToLower(experimentsDetails.Sequence) {
		case "serial":
			if err = injectChaosInSerialMode(experimentsDetails, appVMMoidList, diskIdList, diskPathList, cookie, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
				return err
			}
		case "parallel":
			if err = injectChaosInParallelMode(experimentsDetails, appVMMoidList, diskIdList, diskPathList, cookie, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
				return err
			}
		default:
			return errors.Errorf("%v sequence is not supported", experimentsDetails.Sequence)
		}

		//Waiting for the ramp time after chaos injection
		if experimentsDetails.RampTime != 0 {
			log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.RampTime)
			common.WaitForDuration(experimentsDetails.RampTime)
		}
	}
	return nil
}

//injectChaosInSerialMode will inject the disk loss chaos in serial mode which means one after the other
func injectChaosInSerialMode(experimentsDetails *experimentTypes.ExperimentDetails, appVMMoidList []string, diskIdList []string, diskPathList []string, cookie string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.ChaosDuration {

		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on VM instance"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}
		for i := range diskIdList {

			//Detaching the disk from the vm
			log.Info("[Chaos]: Detaching the disk from the instance")
			if err = vmware.DiskDetach(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie); err != nil {
				return errors.Errorf("disk detachment failed, err: %v", err)
			}

			common.SetTargets(diskIdList[i], "injected", "Disk", chaosDetails)

			//Wait for disk detachment
			log.Infof("[Wait]: Wait for disk detachment for disk %v", diskIdList[i])
			if err = vmware.WaitForDiskDetachment(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				return errors.Errorf("unable to detach the disk from the vm, err: %v", err)
			}

			// run the probes during chaos
			if len(resultDetails.ProbeDetails) != 0 && i == 0 {
				if err = probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
					return err
				}
			}

			//Wait for chaos duration
			log.Infof("[Wait]: Waiting for the chaos interval of %vs", experimentsDetails.ChaosInterval)
			common.WaitForDuration(experimentsDetails.ChaosInterval)

			//Getting the disk attachment status
			diskState, err := vmware.GetDiskState(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie)
			if err != nil {
				return errors.Errorf("failed to get the disk status, err: %v", err)
			}

			switch diskState {
			case "attached":
				log.Info("[Skip]: The disk is already attached")
			default:
				//Attaching the disk to the vm
				log.Info("[Chaos]: Attaching the disk back to the instance")
				if err = vmware.DiskAttach(experimentsDetails.VcenterServer, appVMMoidList[i], diskPathList[i], cookie); err != nil {
					return errors.Errorf("disk attachment failed, err: %v", err)
				}

				//Wait for disk attachment
				log.Infof("[Wait]: Wait for disk attachment for disk %v", diskIdList[i])
				if err = vmware.WaitForDiskAttachment(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
					return errors.Errorf("unable to attach the disk to the vm instance, err: %v", err)
				}
			}
			common.SetTargets(diskIdList[i], "reverted", "Disk", chaosDetails)
		}
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}
	return nil
}

//injectChaosInParallelMode will inject the disk loss chaos in parallel mode that means all at once
func injectChaosInParallelMode(experimentsDetails *experimentTypes.ExperimentDetails, appVMMoidList []string, diskIdList []string, diskPathList []string, cookie string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.ChaosDuration {

		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on vm instance"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		for i := range diskIdList {

			//Detaching the disk from the vm
			log.Info("[Chaos]: Detaching the disk from the vm")
			if err = vmware.DiskDetach(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie); err != nil {
				return errors.Errorf("disk detachment failed, err: %v", err)
			}

			common.SetTargets(diskIdList[i], "injected", "Disk", chaosDetails)
		}

		for i := range diskIdList {

			//Wait for disk detachment
			log.Infof("[Wait]: Wait for disk detachment for disk %v", diskIdList[i])
			if err = vmware.WaitForDiskDetachment(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				return errors.Errorf("unable to detach the disk from the vm, err: %v", err)
			}
		}

		// run the probes during chaos
		if len(resultDetails.ProbeDetails) != 0 {
			if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
				return err
			}
		}

		//Wait for chaos interval
		log.Infof("[Wait]: Waiting for the chaos interval of %vs", experimentsDetails.ChaosInterval)
		common.WaitForDuration(experimentsDetails.ChaosInterval)

		for i := range diskIdList {

			//Getting the disk attachment status
			diskState, err := vmware.GetDiskState(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie)
			if err != nil {
				return errors.Errorf("failed to get the disk status, err: %v", err)
			}

			switch diskState {
			case "attached":
				log.Info("[Skip]: The disk is already attached")
			default:
				//Attaching the disk to the instance
				log.Info("[Chaos]: Attaching the disk to the instance")
				if err = vmware.DiskDetach(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie); err != nil {
					return errors.Errorf("disk attachment failed, err: %v", err)
				}

				//Wait for disk attachment
				log.Infof("[Wait]: Wait for disk attachment for disk %v", diskIdList[i])
				if err = vmware.WaitForDiskAttachment(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
					return errors.Errorf("unable to attach the disk to the vm instance, err: %v", err)
				}
			}
			common.SetTargets(diskIdList[i], "reverted", "Disk", chaosDetails)
		}
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}
	return nil
}

// AbortWatcher will watching for the abort signal and revert the chaos
func AbortWatcher(experimentsDetails *experimentTypes.ExperimentDetails, appVMMoidList, diskIdList []string, diskPathList []string, cookie string, abort chan os.Signal, chaosDetails *types.ChaosDetails) {

	<-abort

	log.Info("[Abort]: Chaos Revert Started")

	for i := range diskIdList {

		//Getting the disk attachment status
		diskState, err := vmware.GetDiskState(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie)
		if err != nil {
			log.Errorf("failed to get the disk state when an abort signal is received, err: %v", err)
		}

		if diskState != "attached" {

			//Wait for disk detachment
			//We first wait for the to get in detached state then we are attaching it.
			log.Info("[Abort]: Wait for complete disk detachment")

			if err = vmware.WaitForDiskDetachment(experimentsDetails.VcenterServer, appVMMoidList[i], diskIdList[i], cookie, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				log.Errorf("unable to detach the disk, err: %v", err)
			}

			//Attaching the disk from the instance
			log.Info("[Chaos]: Attaching the disk from the instance")

			err = vmware.DiskAttach(experimentsDetails.VcenterServer, appVMMoidList[i], diskPathList[i], cookie)
			if err != nil {
				log.Errorf("disk attachment failed when an abort signal is received, err: %v", err)
			}
		}

		common.SetTargets(diskIdList[i], "reverted", "Disk", chaosDetails)
	}

	log.Info("[Abort]: Chaos Revert Completed")
	os.Exit(1)
}
