package vmware

import (
	"bytes"
	"fmt"
	"os/exec"

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
func RebootHost(hostURL, datacenter string) error {

	cmd := fmt.Sprintf("govc host.shutdown -r=true -f=true -dc=%s %s", datacenter, hostURL)
	_, stderr, err := shellout(cmd)

	if err != nil {
		return err
	} else if stderr != "" {
		return errors.Errorf("%s", stderr)
	}

	return nil
}
