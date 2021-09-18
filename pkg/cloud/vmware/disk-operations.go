package vmware

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// GetDisks returns the list of ids of disks attached to a given VM
func GetDisks(vcenterServer, vmId, cookie string) ([]string, error) {

	type DiskList struct {
		MsgValue []struct {
			MsgDisk string `json:"disk"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm/"+vmId+"/hardware/disk/", nil)
	if err != nil {
		return nil, errors.Errorf(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return nil, errors.Errorf("error during the fetching of disks: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var diskDetails DiskList
	json.Unmarshal(body, &diskDetails)

	var diskIdList []string
	for _, disk := range diskDetails.MsgValue {
		diskIdList = append(diskIdList, disk.MsgDisk)
	}

	return diskIdList, nil
}
