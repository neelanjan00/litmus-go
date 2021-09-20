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

// getHost returns the host id, host connection state, and host power state for a given host name
func getHost(vcenterServer, hostName, cookie string) (string, string, string, error) {

	type Host struct {
		MsgValue []struct {
			MsgHost            string `json:"host"`
			MsgConnectionState string `json:"connection_state"`
			MsgPowerState      string `json:"power_state"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/host?filter.names.1="+hostName, nil)
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

	if len(hostDetails.MsgValue) == 0 {
		return "", "", "", errors.Errorf("%s host not found", hostName)
	}

	return hostDetails.MsgValue[0].MsgHost, hostDetails.MsgValue[0].MsgConnectionState, hostDetails.MsgValue[0].MsgPowerState, nil
}

// GetHostDetails checks if the given host is powered on and connected, and later returns the host id
func HostStatusCheck(vcenterServer, hostName, datacenter, cookie string) (string, error) {

	if vcenterServer == "" {
		return "", errors.Errorf("no vcenter server provided, please provide the server url")
	}

	if hostName == "" {
		return "", errors.Errorf("no host name provided, please provide the target host name")
	}

	if datacenter == "" {
		return "", errors.Errorf("no datacenter provided, please provide the datacenter name of the target host")
	}

	hostId, connectionState, powerState, err := getHost(vcenterServer, hostName, cookie)
	if err != nil {
		return "", err
	}

	if connectionState != "CONNECTED" {
		return "", errors.Errorf("host not in CONNECTED state")
	}

	if powerState != "POWERED_ON" {
		return "", errors.Errorf("host not in POWERED_ON state")
	}

	return hostId, nil
}

// GetHostConnectionStatus returns the connection status of the given host i.e. CONNECTED, DISCONNECTED, or NOT_RESPONDING
func GetHostConnectionStatus(vcenterServer, hostName, cookie string) (string, error) {

	_, connectionState, _, err := getHost(vcenterServer, hostName, cookie)
	if err != nil {
		return "", err
	}

	return connectionState, nil
}

// GetVMDetails returns two lists of VM ids of powered-on VMs and powered-off or suspended VMs that are attached to a given host
func GetVMDetails(vcenterServer, hostId, cookie string) ([]string, []string, error) {

	type VM struct {
		MsgValue []struct {
			MsgVm         string `json:"vm"`
			MsgPowerState string `json:"power_state"`
		} `json:"value"`
	}

	var poweredOnVMList, poweredOffOrSuspendedVMList []string

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm?filter.hosts.1="+hostId, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return nil, nil, errors.Errorf("error during the fetching of VM details: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var vmDetails VM
	json.Unmarshal(body, &vmDetails)

	for _, vm := range vmDetails.MsgValue {
		if vm.MsgPowerState == "POWERED_ON" {
			poweredOnVMList = append(poweredOnVMList, vm.MsgVm)
		} else {
			poweredOffOrSuspendedVMList = append(poweredOffOrSuspendedVMList, vm.MsgVm)
		}
	}

	return poweredOnVMList, poweredOffOrSuspendedVMList, nil
}
