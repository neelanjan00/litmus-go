package vmware

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ErrorResponse struct {
	MsgValue struct {
		MsgMessages []struct {
			MsgDefaultMessage string `json:"default_message"`
		} `json:"messages"`
	} `json:"value"`
}

// VirtualDiskDetach will detach a virtual disk from a VM
func VirtualDiskDetach(vcenterServer, appVMMoid, diskId, cookie string) error {

	req, err := http.NewRequest("DELETE", "https://"+vcenterServer+"/rest/vcenter/vm/"+appVMMoid+"/hardware/disk/"+diskId, nil)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errors.Errorf(err.Error())
		}

		json.Unmarshal(body, &errorResponse)

		return errors.Errorf("error during disk detachment: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	log.InfoWithValues("Detached disk having:", logrus.Fields{
		"VM ID":   appVMMoid,
		"Disk ID": diskId,
	})

	return nil
}

// VirtualDiskAttach will attach a virtual disk to a VM
func VirtualDiskAttach(vcenterServer, appVMMoid, diskPath, cookie string) error {

	type AttachDiskResponse struct {
		MsgValue string `json:"value"`
	}

	jsonString := fmt.Sprintf(`{"spec":{"backing":{"type":"VMDK_FILE","vmdk_file":"%s"}}}`, diskPath)

	req, err := http.NewRequest("POST", "https://"+vcenterServer+"/rest/vcenter/vm/"+appVMMoid+"/hardware/disk", bytes.NewBuffer([]byte(jsonString)))
	if err != nil {
		return errors.Errorf(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errors.Errorf(err.Error())
		}

		json.Unmarshal(body, &errorResponse)

		return errors.Errorf("error during disk attachment: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var response AttachDiskResponse
	json.Unmarshal(body, &response)

	log.InfoWithValues("Attached disk having:", logrus.Fields{
		"VM ID":   appVMMoid,
		"Disk ID": response.MsgValue,
	})

	return nil
}
