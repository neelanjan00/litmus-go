package vmware

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type ErrorResponse struct {
	MsgValue struct {
		MsgMessages []struct {
			MsgDefaultMessage string `json:"default_message"`
		} `json:"messages"`
	} `json:"value"`
}

// GetHostDetails returns the host id, host connection state, and host power state for a given host URL
func GetHostDetails(vcenterServer, hostURL, cookie string) (string, string, string, error) {

	type Host struct {
		MsgValue []struct {
			MsgHost            string `json:"host"`
			MsgConnectionState string `json:"connection_state"`
			MsgPowerState      string `json:"power_state"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/host?filter.names.1="+hostURL, nil)
	if err != nil {
		return "", "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return "", "", "", errors.Errorf("error during the fetching of host details: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var hostDetails Host
	json.Unmarshal(body, &hostDetails)

	return hostDetails.MsgValue[0].MsgHost, hostDetails.MsgValue[0].MsgConnectionState, hostDetails.MsgValue[0].MsgPowerState, nil
}

// GetPoweredOnVMDetails returns the VM ids that are in POWERED_ON state and are attached to a given host
func GetPoweredOnVMDetails(vcenterServer, host, cookie string) ([]string, error) {

	type VM struct {
		MsgValue []struct {
			MsgVm string `json:"vm"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm?filter.hosts.1="+host+"&filter.power_states.1=POWERED_ON", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return nil, errors.Errorf("error during the fetching of VM details: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var vmDetails VM
	json.Unmarshal(body, &vmDetails)

	var vmList []string
	for _, vm := range vmDetails.MsgValue {
		vmList = append(vmList, vm.MsgVm)
	}

	return vmList, nil
}
