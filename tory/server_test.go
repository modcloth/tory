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
	testHost           *host
	testHostJSONReader io.Reader
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	testServer = buildServer(":9999", os.Getenv("DATABASE_URL"),
		"public", `/ansible/hosts/test`, false)
	testHost = newHost()
	testHost.Name = fmt.Sprintf("test%d.example.com", rand.Intn(255))
	testHost.IP = fmt.Sprintf("10.10.1.%d", rand.Intn(255))
	testHostJSONBytes, err := json.Marshal(map[string]*host{"host": testHost})
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
}
