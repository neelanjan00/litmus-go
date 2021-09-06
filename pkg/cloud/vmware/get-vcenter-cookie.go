package vmware

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// Message contains attribute for message
type Message struct {
	MsgValue string `json:"value"`
}

//GetVcenterSessionID returns the vcenter sessionid
func GetVcenterSessionID(vcenterServer, vcenterUser, vcenterPass string) (string, error) {

	//Leverage Go's HTTP Post function to make request
	req, err := http.NewRequest("POST", "https://"+vcenterServer+"/rest/com/vmware/cis/session", nil)
	if err != nil {
		return "", errors.Errorf(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(vcenterUser, vcenterPass)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	//Handle Error
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

		return "", errors.Errorf("error during authentication: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var m Message
	json.Unmarshal(body, &m)

	login := "vmware-api-session-id=" + m.MsgValue + ";Path=/rest;Secure;HttpOnly"
	return login, nil
}
