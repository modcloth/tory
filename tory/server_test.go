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
	testAuth   string
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

type varValue struct {
	Value string `json:"value"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	testAuth = fmt.Sprintf("secrety-secret-%d", rand.Int())
	testServer = buildServer(&ServerOptions{
		Addr:        ":9999",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		StaticDir:   "public",
		AuthToken:   testAuth,
		Prefix:      `/ansible/hosts/test`,
	})
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

func makeRequest(method, urlStr string, body io.Reader, auth string) *httptest.ResponseRecorder {
	return makeRequestWithHeaders(method, urlStr, body, http.Header{"Authorization": []string{fmt.Sprintf("token %s", auth)}})
}

func makeRequestWithHeaders(method, urlStr string, body io.Reader, headers http.Header) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		panic(err)
	}

	req.Header = headers

	w := httptest.NewRecorder()
	testServer.n.ServeHTTP(w, req)

	return w
}

func mustCreateHost(t *testing.T) *HostJSON {
	h, reader := getTestHostJSONReader()

	w := makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader, testAuth)
	if w.Code != 201 {
		t.Fatalf("response code is not 201: %v", w.Code)
	}

	return h
}

func TestHandleGZIPEncoding(t *testing.T) {
	w := makeRequestWithHeaders("GET", `/ping`, nil, http.Header{"Accept-Encoding": []string{"gzip"}})

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("response was not gzip encoded")
	}
}

func TestHandlePing(t *testing.T) {
	w := makeRequest("GET", `/ping`, nil, "")
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	if w.Body.String() != "PONG\n" {
		t.Fatalf("body is not \"PONG\"")
	}
}

func TestHandleDebugVars(t *testing.T) {
	w := makeRequest("GET", `/debug/vars`, nil, "")
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
	h := mustCreateHost(t)

	for _, s := range []string{
		`/ansible/hosts/test`,
		fmt.Sprintf(`/ansible/hosts/test?since=%s`, time.Now().Add(-24*time.Hour).Format(time.RFC3339)),
		fmt.Sprintf(`/ansible/hosts/test?before=%s`, time.Now().Add(24*time.Hour).Format(time.RFC3339)),
		fmt.Sprintf(`/ansible/hosts/test?since=%s&before=%s`,
			time.Now().Add(-24*time.Hour).Format(time.RFC3339),
			time.Now().Add(24*time.Hour).Format(time.RFC3339)),
		`/ansible/hosts/test?vars-only=`,
	} {
		w := makeRequest("GET", s, nil, "")
		if w.Code != 200 {
			t.Fatalf("GET %s did not return 200: %v", s, w.Code)
		}

		inv := newInventory()
		err := json.NewDecoder(w.Body).Decode(inv)
		if err != nil {
			t.Error(err)
		}

		if inv.Meta == nil {
			t.Fatalf("GET %s: body does not contain \"_meta\"", s)
		}

		if inv.Meta.Hostvars == nil {
			t.Fatalf("GET %s: body meta does not contain \"hostvars\"", s)
		}

		if _, ok := inv.Meta.Hostvars[h.Name]; !ok {
			t.Fatalf("GET %s: body meta does not contain \"hostvars\" by name", s)
		}
	}
}

func TestHandleGetHost(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("GET", `/ansible/hosts/test/`+h.Name, nil, "")
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

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name+`?vars-only=1`, nil, "")
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

	w := makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader, testAuth)
	if w.Code != 201 {
		t.Fatalf("response code is not 201: %v", w.Code)
	}

	hj, err := hostJSONFromHTTPBody(w.Body)
	if err != nil {
		t.Error(err)
	}

	h.ID = hj.ID
	delete(h.Tags, "role")

	newIP := fmt.Sprintf("10.10.3.%d", rand.Intn(255))
	h.IP = newIP
	reader = getReaderForHost(h)

	w = makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader, testAuth)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	hj, err = hostJSONFromHTTPBody(w.Body)
	if err != nil {
		t.Error(err)
	}

	if _, ok := hj.Tags["role"]; !ok {
		t.Fatalf("role tag was not retained on update, tags=%#v", hj.Tags)
	}

	if hj.ID != h.ID {
		t.Fatalf("outgoing id does not match: %v != %v", hj.ID, h.ID)
	}

	if hj.Name != h.Name {
		t.Fatalf("outgoing hostname does not match: %v != %v", hj.Name, h.Name)
	}

	w = makeRequest("GET", `/ansible/hosts/test`, nil, "")
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	inv := newInventory()
	err = json.NewDecoder(w.Body).Decode(inv)

	if g := inv.GetGroup(h.IP); g == nil {
		t.Fatalf("response does not contain IP as group")
	}

	tagTeamGroup := inv.GetGroup(fmt.Sprintf("tag_team_fribbles"))
	if tagTeamGroup == nil {
		t.Fatalf("response does not contain tag team group")
	}

	hasHostname := false
	for _, hostname := range tagTeamGroup {
		if hostname == h.Name {
			hasHostname = true
		}
	}

	if !hasHostname {
		t.Fatalf("test host %q not in tag team group", h.Name)
	}

	typeGroup := inv.GetGroup(fmt.Sprintf("type_virtualmachine"))
	if typeGroup == nil {
		t.Fatalf("response does not contain type group")
	}

	hasHostname = false
	for _, hostname := range typeGroup {
		if hostname == h.Name {
			hasHostname = true
		}
	}

	if !hasHostname {
		t.Fatalf("test host %q not in tag team group", h.Name)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name, nil, "")
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

	origPackage := h.Package
	origType := h.Type
	origImage := h.Image

	h.Package = ""
	h.Type = ""
	h.Image = ""
	reader = getReaderForHost(h)

	w = makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader, testAuth)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	hj, err = hostJSONFromHTTPBody(w.Body)
	if err != nil {
		t.Error(err)
	}

	if hj.Package == "" || hj.Package != origPackage {
		t.Fatalf("package was overwritten by empty string: %q != %q", hj.Package, origPackage)
	}

	if hj.Type == "" || hj.Type != origType {
		t.Fatalf("type was overwritten by empty string: %q != %q", hj.Type, origType)
	}

	if hj.Image == "" || hj.Image != origImage {
		t.Fatalf("image was overwritten by empty string: %q != %q", hj.Package, origImage)
	}
}

func TestHandleUpdateHostUnauthorized(t *testing.T) {
	h, reader := getTestHostJSONReader()

	w := makeRequest("PUT", `/ansible/hosts/test/`+h.Name, reader, "bogus")
	if w.Code != 401 {
		t.Fatalf("response code is not 401: %v", w.Code)
	}
}

func TestHandleDeleteHost(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("DELETE", `/ansible/hosts/test/`+h.Name, nil, testAuth)
	if w.Code != 204 {
		t.Fatalf("response code is not 204: %v", w.Code)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name, nil, "")
	if w.Code != 404 {
		t.Fatalf("response code is not 404: %v", w.Code)
	}
}

func TestHandleDeleteHostUnauthorized(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("DELETE", `/ansible/hosts/test/`+h.Name, nil, "bogus")
	if w.Code != 401 {
		t.Fatalf("response code is not 401: %v", w.Code)
	}
}

func TestHandleGetHostVar(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("GET", `/ansible/hosts/test/`+h.Name+`/vars/memory`, nil, "")
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	v := &varValue{}
	err := json.NewDecoder(w.Body).Decode(v)
	if err != nil {
		t.Error(err)
	}

	if v.Value != h.Vars["memory"] {
		t.Fatalf("outgoing memory does not match: %s != %s", v.Value, h.Vars["memory"])
	}
}

func TestHandleUpdateHostVar(t *testing.T) {
	h := mustCreateHost(t)
	b, err := json.Marshal(&varValue{Value: "1024"})
	if err != nil {
		t.Error(err)
	}

	r := bytes.NewReader(b)
	w := makeRequest("PUT", `/ansible/hosts/test/`+h.Name+`/vars/memory`, r, testAuth)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name+`/vars/memory`, nil, "")
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	v := &varValue{}
	err = json.NewDecoder(w.Body).Decode(v)
	if err != nil {
		t.Error(err)
	}

	if v.Value != "1024" {
		t.Fatalf("outgoing memory does not match: %s != 1024", v.Value)
	}
}

func TestHandleDeleteHostVar(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("DELETE", `/ansible/hosts/test/`+h.Name+`/vars/memory`, nil, testAuth)
	if w.Code != 204 {
		t.Fatalf("response code is not 204: %v", w.Code)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name+`/vars/memory`, nil, "")
	if w.Code != 404 {
		t.Fatalf("response code is not 404: %v", w.Code)
	}
}

func TestHandleGetHostTag(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("GET", `/ansible/hosts/test/`+h.Name+`/tags/team`, nil, "")
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	v := &varValue{}
	err := json.NewDecoder(w.Body).Decode(v)
	if err != nil {
		t.Error(err)
	}

	if v.Value != h.Tags["team"] {
		t.Fatalf("outgoing team does not match: %s != %s", v.Value, h.Tags["team"])
	}
}

func TestHandleUpdateHostTag(t *testing.T) {
	h := mustCreateHost(t)
	b, err := json.Marshal(&varValue{Value: "sploop"})
	if err != nil {
		t.Error(err)
	}

	r := bytes.NewReader(b)
	w := makeRequest("PUT", `/ansible/hosts/test/`+h.Name+`/tags/team`, r, testAuth)
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name+`/tags/team`, nil, "")
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	v := &varValue{}
	err = json.NewDecoder(w.Body).Decode(v)
	if err != nil {
		t.Error(err)
	}

	if v.Value != "sploop" {
		t.Fatalf("outgoing team does not match: %s != sploop", v.Value)
	}
}

func TestHandleDeleteHostTag(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("DELETE", `/ansible/hosts/test/`+h.Name+`/tags/team`, nil, testAuth)
	if w.Code != 204 {
		t.Fatalf("response code is not 204: %v", w.Code)
	}

	w = makeRequest("GET", `/ansible/hosts/test/`+h.Name+`/tags/team`, nil, "")
	if w.Code != 404 {
		t.Fatalf("response code is not 404: %v", w.Code)
	}
}

func TestHandleFilterHosts(t *testing.T) {
	h := mustCreateHost(t)

	w := makeRequest("GET", `/ansible/hosts/test?name=`+h.Name, nil, "")
	if w.Code != 200 {
		t.Fatalf("response code is not 200: %v", w.Code)
	}

	res := map[string]json.RawMessage{}
	err := json.NewDecoder(w.Body).Decode(&res)

	if err != nil {
		t.Error(err)
	}

	ipGroup, ok := res[h.IP]
	if !ok {
		t.Fatalf("ip group not present")
	}

	ipGroupSlice := []string{}
	err = json.Unmarshal(ipGroup, &ipGroupSlice)
	if err != nil {
		t.Error(err)
	}

	if len(ipGroupSlice) == 0 {
		t.Fatalf("ip group is empty")
	}
}
