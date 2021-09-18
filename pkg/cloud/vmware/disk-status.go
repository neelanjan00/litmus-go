package vmware

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GetDiskState will verify if the given disk is attached to the given VM or not
func GetDiskState(vcenterServer, vmId, diskId, cookie string) (string, error) {

	type DiskList struct {
		MsgValue []struct {
			MsgDisk string `json:"disk"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm/"+vmId+"/hardware/disk/", nil)
	if err != nil {
		return "", errors.Errorf(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Errorf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return "", errors.Errorf("error during disk state fetch: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var diskList DiskList
	json.Unmarshal(body, &diskList)

	for _, disk := range diskList.MsgValue {

		if disk.MsgDisk == diskId {

			log.InfoWithValues("The selected disk is:", logrus.Fields{
				"VM ID":   vmId,
				"Disk ID": diskId,
			})

			return "attached", nil
		}
	}

	return "detached", nil
}
