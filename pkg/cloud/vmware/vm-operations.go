package vmware

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
)

//StartVM starts the VMWare VM
func StartVM(vcenterServer, vmId, cookie string) error {

	req, err := http.NewRequest("POST", "https://"+vcenterServer+"/rest/vcenter/vm/"+vmId+"/power/start", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf(err.Error())
		}

		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return errors.Errorf("failed to start the VM: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	return nil
}

//StopVM stops the desired VMWare VM
func StopVM(vcenterServer, vmId, cookie string) error {

	req, err := http.NewRequest("POST", "https://"+vcenterServer+"/rest/vcenter/vm/"+vmId+"/power/stop", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf(err.Error())
		}

		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return errors.Errorf("failed to stop the VM: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	return nil
}

func WaitForVMStart(timeout, delay int, vcenterServer, vmId, cookie string) error {
	log.Infof("[Status]: Checking %s VM power status", vmId)
	return retry.Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {
			vmPowerState, err := GetVMStatus(vcenterServer, vmId, cookie)
			if err != nil {
				return errors.Errorf("failed to get the VM power state: %s", err.Error())
			}
			if vmPowerState != "POWERED_ON" {
				log.Infof("The VM power state is %s", vmPowerState)
				return errors.Errorf("vm is not yet in POWERED_ON state")
			}
			log.Infof("The VM power state is %s", vmPowerState)
			return nil
		})
}
