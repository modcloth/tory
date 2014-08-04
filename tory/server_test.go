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

	"github.com/bitly/go-simplejson"
)

var (
	testServer         *server
	testHost           *hostJSON
	testHostJSONReader io.Reader
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	testServer = buildServer(":9999", os.Getenv("DATABASE_URL"),
		"public", `/ansible/hosts/test`, false)

	testHost = &hostJSON{
		Name:    fmt.Sprintf("test%d.example.com", rand.Intn(255)),
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

	testHostJSONBytes, err := json.Marshal(map[string]*hostJSON{"host": testHost})
	if err != nil {
		panic(err)
	}

	testHostJSONReader = bytes.NewReader(testHostJSONBytes)
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

	j, err := simplejson.NewFromReader(w.Body)
	if err != nil {
		t.Fatalf("response is not json: %v", w.Body.String())
	}

	if _, ok := j.CheckGet("env"); !ok {
		t.Fatalf("body does not contain \"env\"")
	}
}

func TestHandleGetHostInventory(t *testing.T) {
	w := makeRequest("GET", `/ansible/hosts/test`, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	j, err := simplejson.NewFromReader(w.Body)
	if err != nil {
		t.Fatalf("response is not json: %v", w.Body.String())
	}

	if _, ok := j.CheckGet("_meta"); !ok {
		t.Fatalf("body does not contain \"_meta\"")
	}
}

func TestHandleAddHostToInventory(t *testing.T) {
	w := makeRequest("POST", `/ansible/hosts/test`, testHostJSONReader)
	if w.Code != 201 {
		t.Fatalf("response code is not 201: %v", w.Code)
	}

	j, err := simplejson.NewFromReader(w.Body)
	if err != nil {
		t.Fatalf("response is not json: %v", w.Body.String())
	}

	host, ok := j.CheckGet("host")
	if !ok {
		t.Fatalf("body does not contain \"host\"")
	}

	_, ok = host.CheckGet("id")
	if !ok {
		t.Fatalf("body does not contain \"host.id\"")
	}

	hostname, ok := host.CheckGet("name")
	if !ok {
		t.Fatalf("body does not contain \"host.name\"")
	}

	hostnameString, err := hostname.String()
	if err != nil {
		t.Error(err)
		return
	}

	if hostnameString != testHost.Name {
		t.Fatalf("returned hostname does not match: %v != %v", hostname, testHost.Name)
	}

	w = makeRequest("GET", `/ansible/hosts/test`, nil)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	j, err = simplejson.NewFromReader(w.Body)
	if err != nil {
		t.Fatalf("response is not json: %v", w.Body.String())
	}

	if _, ok := j.CheckGet(testHost.Name); !ok {
		t.Fatalf("response does not contain host name as group")
	}

	tagTeamGroup, ok := j.CheckGet(fmt.Sprintf("tag_team_fribbles"))
	if !ok {
		t.Fatalf("response does not contain tag team group")
	}

	ips, err := tagTeamGroup.StringArray()
	if err != nil {
		t.Fatalf("failed to get ip addresses in team group")
	}

	hasIP := false
	for _, ip := range ips {
		if ip == testHost.IP {
			hasIP = true
		}
	}

	if !hasIP {
		t.Fatalf("test host ip %q not in tag team group", testHost.IP)
	}

	typeGroup, ok := j.CheckGet(fmt.Sprintf("type_virtualmachine"))
	if !ok {
		t.Fatalf("response does not contain type group")
	}

	ips, err = typeGroup.StringArray()
	if err != nil {
		t.Fatalf("failed to get ip addresses in type group")
	}

	hasIP = false
	for _, ip := range ips {
		if ip == testHost.IP {
			hasIP = true
		}
	}

	if !hasIP {
		t.Fatalf("test host ip %q not in tag team group", testHost.IP)
	}
}
