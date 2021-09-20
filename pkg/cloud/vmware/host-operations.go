package vmware

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
)

// shellout executes a given shell command and returns the stdout, stderr, and execution error
func shellout(command string) (string, string, error) {

	var stdout, stderr bytes.Buffer

	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

// RebootHost causes a given host in a particular datacenter to reboot
func RebootHost(hostName, datacenter string) error {

	cmd := fmt.Sprintf("govc host.shutdown -r=true -f=true -dc=%s %s", datacenter, hostName)
	_, stderr, err := shellout(cmd)

	if err != nil {
		return err
	} else if stderr != "" {
		return errors.Errorf("%s", stderr)
	}

	return nil
}

// WaitForHostToDisconnect will wait for the host to attain the NOT_RESPONDING status
func WaitForHostToDisconnect(timeout, delay int, vcenterServer, hostName, cookie string) error {

	log.Info("[Status]: Checking host connection status")
	return retry.Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			hostState, err := GetHostConnectionStatus(vcenterServer, hostName, cookie)
			if err != nil {
				return errors.Errorf("failed to get the host connection status: %s", err.Error())
			}

			if hostState != "NOT_RESPONDING" {
				log.Infof("[Info]: The host connection state is %s", hostState)
				return errors.Errorf("host is not yet in NOT_RESPONDING state")
			}

			log.Infof("[Info]: The host connection state is %s", hostState)
			return nil
		})
}

// WaitForHostToConnect will wait for the host to attain the CONNECTED status
func WaitForHostToConnect(timeout, delay int, vcenterServer, hostName, cookie string) error {

	log.Info("[Status]: Checking host connection status")
	return retry.Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			hostState, err := GetHostConnectionStatus(vcenterServer, hostName, cookie)
			if err != nil {
				return errors.Errorf("failed to get the host connection status: %s", err.Error())
			}

			if hostState != "CONNECTED" {
				log.Infof("[Info]: The host connection state is %s", hostState)
				return errors.Errorf("host is not yet in CONNECTED state")
			}

			log.Infof("[Info]: The host connection state is %s", hostState)
			return nil
		})
}
