package vmware

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WaitForDiskDetachment will wait for the disk to completely detach from the VM
func WaitForDiskDetachment(vcenterServer, appVMMoid, diskId, cookie string, delay, timeout int) error {

	log.Info("[Status]: Checking disk status for detachment")
	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			diskState, err := GetDiskState(vcenterServer, appVMMoid, diskId, cookie)
			if err != nil {
				return errors.Errorf("failed to get the disk state")
			}

			if diskState != "detached" {
				log.Infof("[Info]: The disk state is %v", diskState)
				return errors.Errorf("disk is not yet in detached state")
			}

			log.Infof("[Info]: The disk state is %v", diskState)
			return nil
		})
}

// WaitForDiskAttachment will wait for the disk to get attached to the VM
func WaitForDiskAttachment(vcenterServer, appVMMoid, diskId, cookie string, delay, timeout int) error {

	log.Info("[Status]: Checking disk status for attachment")
	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			diskState, err := GetDiskState(vcenterServer, appVMMoid, diskId, cookie)
			if err != nil {
				return errors.Errorf("failed to get the disk status")
			}

			if diskState != "attached" {
				log.Infof("[Info]: The disk state is %v", diskState)
				return errors.Errorf("disk is not yet in attached state")
			}

			log.Infof("[Info]: The disk state is %v", diskState)
			return nil
		})
}

// GetDiskState will verify if the given disk is attached to the given VM or not
func GetDiskState(vcenterServer, appVMMoid, diskId, cookie string) (string, error) {

	type DiskList struct {
		MsgValue []struct {
			MsgDisk string `json:"disk"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm/"+appVMMoid+"/hardware/disk/", nil)
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
				"VM ID":   appVMMoid,
				"Disk ID": diskId,
			})

			return "attached", nil
		}
	}

	return "detached", nil
}

//DiskStateCheck will check the attachment state of the given disks
func DiskStateCheck(vcenterServer, appVMMoids, diskIds, cookie string) error {

	if vcenterServer == "" {
		return errors.Errorf("no vcenter server provided, please provide the server url")
	}

	diskIdList := strings.Split(diskIds, ",")
	if len(diskIdList) == 0 {
		return errors.Errorf("no disk id provided, please provide disk id")
	}

	appVMMoidList := strings.Split(appVMMoids, ",")
	if len(appVMMoidList) == 0 {
		return errors.Errorf("no vm id provided, please provide vm id")
	}

	if len(diskIdList) != len(appVMMoidList) {
		return errors.Errorf("unequal number of disk ids and vm ids found, please verify the input details")
	}

	for i := range diskIdList {

		diskState, err := GetDiskState(vcenterServer, appVMMoidList[i], diskIdList[i], cookie)

		if err != nil {
			return errors.Errorf("failed to get the disk %v in attached state, err: %v", diskIdList[i], err.Error())
		}

		if diskState != "attached" {
			return errors.Errorf("%v disk state check failed, disk is in detached state", diskIdList[i])
		}
	}

	return nil
}
