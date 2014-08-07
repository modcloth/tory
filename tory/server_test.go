package tory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var (
	testServer *server
)

type debugVars struct {
	Env map[string]string `json:"env"`
}

type hostVars struct {
	Team        string `json:"team"`
	Role        string `json:"role"`
	Provisioner string `json:"provisioner"`
	Memory      string `json:"memory"`
	Disk        string `json:"disk"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	testServer = buildServer(":9999", os.Getenv("DATABASE_URL"),
		"public", `/ansible/hosts/test`, false)

}

func getTestHostJSONReader() (*HostJSON, io.Reader) {
	testHost := &HostJSON{
		Name:    fmt.Sprintf("test%d-%d.example.com", rand.Intn(16384), time.Now().UTC().UnixNano()),
		IP:      fmt.Sprintf("10.10.1.%d", rand.Intn(255)),
		Package: "fancy-town-80",
		Image:   "ubuntu-14.04",
		Type:    "virtualmachine",
		Tags: map[string]interface{}{
			"team":        "fribbles",
			"env":         "prod",
			"role":        "job",
			"provisioner": "p.freely",
		},
		Vars: map[string]interface{}{
			"memory": "512",
			"disk":   "16384",
		},
	}

	return testHost, getReaderForHost(testHost)
}

func getReaderForHost(testHost *HostJSON) io.Reader {
	testHostJSONBytes, err := json.Marshal(&HostPayload{testHost})
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(testHostJSONBytes)
}

func makeRequest(method, urlStr string, body io.Reader) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	testServer.n.ServeHTTP(w, req)

	return w
}

func TestHandlePing(t *testing.T) {
	w := makeRequest("GET", `/ping`, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	if w.Body.String() != "PONG\n" {
		t.Fatalf("body is not \"PONG\"")
	}
}

func TestHandleDebugVars(t *testing.T) {
	w := makeRequest("GET", `/debug/vars`, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	dv := &debugVars{}
	err := json.NewDecoder(w.Body).Decode(dv)
	if err != nil {
		t.Error(err)
	}

	if dv.Env == nil {
		t.Fatalf("body does not contain \"env\"")
	}

	if _, ok := dv.Env["DATABASE_URL"]; !ok {
		t.Fatalf("env does not contain whitelisted vars")
	}
}

func TestHandleGetHostInventory(t *testing.T) {
	w := makeRequest("GET", `/ansible/hosts/test`, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	inv := newInventory()
	err := json.NewDecoder(w.Body).Decode(inv)
	if err != nil {
		t.Error(err)
	}

	if inv.Meta == nil {
		t.Fatalf("body does not contain \"_meta\"")
	}

	if inv.Meta.Hostvars == nil {
		t.Fatalf("body meta does not contain \"hostvars\"")
	}
}

func TestHandleGetHost(t *testing.T) {
	h, reader := getTestHostJSONReader()

	w := makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader)
	if w.Code != 201 {
		t.Fatalf("response code is not 201: %v", w.Code)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	hj, err := hostJSONFromHTTPBody(w.Body)
	if err != nil {
		t.Error(err)
	}

	if hj.IP != h.IP {
		t.Fatalf("outgoing ip addr does not match: %s != %s", hj.IP, h.IP)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name+`?vars-only=1`, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	hv := &hostVars{}
	err = json.NewDecoder(w.Body).Decode(hv)
	if err != nil {
		t.Error(err)
	}

	if hv.Team != h.Tags["team"] {
		t.Fatalf("outgoing team does not match: %s != %s", hv.Team, h.Tags["team"])
	}

	if hv.Role != h.Tags["role"] {
		t.Fatalf("outgoing role does not match: %s != %s", hv.Role, h.Tags["role"])
	}

	if hv.Provisioner != h.Tags["provisioner"] {
		t.Fatalf("outgoing role does not match: %s != %s", hv.Provisioner, h.Tags["role"])
	}

	if hv.Memory != h.Vars["memory"] {
		t.Fatalf("outgoing memory does not match: %s != %s", hv.Memory, h.Vars["memory"])
	}

	if hv.Disk != h.Vars["disk"] {
		t.Fatalf("outgoing disk does not match: %s != %s", hv.Disk, h.Vars["disk"])
	}
}

func TestHandleUpdateHost(t *testing.T) {
	h, reader := getTestHostJSONReader()

	w := makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader)
	if w.Code != 201 {
		t.Fatalf("response code is not 201: %v", w.Code)
	}

	hj, err := hostJSONFromHTTPBody(w.Body)
	if err != nil {
		t.Error(err)
	}

	h.ID = hj.ID

	newIP := fmt.Sprintf("10.10.3.%d", rand.Intn(255))
	h.IP = newIP
	reader = getReaderForHost(h)

	w = makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	hj, err = hostJSONFromHTTPBody(w.Body)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%#v\n", hj)

	if hj.ID != h.ID {
		t.Fatalf("outgoing id does not match: %v != %v", hj.ID, h.ID)
	}

	if hj.Name != h.Name {
		t.Fatalf("outgoing hostname does not match: %v != %v", hj.Name, h.Name)
	}

	w = makeRequest("GET", `/ansible/hosts/test`, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	inv := newInventory()
	err = json.NewDecoder(w.Body).Decode(inv)

	if g := inv.GetGroup(h.Name); g == nil {
		t.Fatalf("response does not contain host name as group")
	}

	tagTeamGroup := inv.GetGroup(fmt.Sprintf("tag_team_fribbles"))
	if tagTeamGroup == nil {
		t.Fatalf("response does not contain tag team group")
	}

	hasIP := false
	for _, ip := range tagTeamGroup {
		if ip == h.IP {
			hasIP = true
		}
	}

	if !hasIP {
		t.Fatalf("test host ip %q not in tag team group", h.IP)
	}

	typeGroup := inv.GetGroup(fmt.Sprintf("type_virtualmachine"))
	if typeGroup == nil {
		t.Fatalf("response does not contain type group")
	}

	hasIP = false
	for _, ip := range typeGroup {
		if ip == h.IP {
			hasIP = true
		}
	}

	if !hasIP {
		t.Fatalf("test host ip %q not in tag team group", h.IP)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	hj, err = hostJSONFromHTTPBody(w.Body)
	if err != nil {
		t.Error(err)
	}

	if hj.IP != newIP {
		t.Fatalf("ip address was not updated: %s != %s", hj.IP, newIP)
	}
}
