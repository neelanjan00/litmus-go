package vmware

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

//GetVMStatus gets the current status of Vcenter VM
func GetVMStatus(vcenterServer, vmId, cookie string) (string, error) {

	type VM struct {
		MsgValue struct {
			MsgState string `json:"state"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm/"+vmId+"/power/", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return "", errors.Errorf("failed to fetch VM status: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var vmDetails VM
	json.Unmarshal(body, &vmDetails)

	return vmDetails.MsgValue.MsgState, nil
}
