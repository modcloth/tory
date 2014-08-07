package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RequestJSON struct {
	Name string            `json:"name"`
	IP   string            `json:"ip"`
	Tags map[string]string `json:"tags,omitempty"`
	Vars map[string]string `json:"vars,omitempty"`

	ToryServer string `json:"-"`
}

func PutHost(hj *RequestJSON) (error, int) {
	hjBytes, err := json.Marshal(map[string]*RequestJSON{"host": hj})
	if err != nil {
		return err, 500
	}

	buf := bytes.NewReader(hjBytes)
	req, err := http.NewRequest("PUT", hj.ToryServer+"/"+hj.Name, buf)
	if err != nil {
		return err, 500
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err, 500
	}

	return nil, resp.StatusCode
}

func StringifiedMap(m map[string]interface{}) map[string]string {
	out := map[string]string{}

	for key, value := range m {
		out[key] = fmt.Sprintf("%v", value)
	}

	return out
}
