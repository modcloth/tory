package tory

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bitly/go-simplejson"
)

var (
	testServer *server
)

func init() {
	testServer = buildServer(":9999", os.Getenv("DATABASE_URL"),
		"public", `/ansible/hosts/test`, false)
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
		t.Fail()
	}
	if w.Body.String() != "PONG\n" {
		t.Fail()
	}
}

func TestHandleDebugVars(t *testing.T) {
	w := makeRequest("GET", `/debug/vars`, nil)
	if w.Code != 200 {
		t.Fail()
	}

	j, err := simplejson.NewFromReader(w.Body)
	if err != nil {
		t.Fail()
	}

	if _, ok := j.CheckGet("env"); !ok {
		t.Fail()
	}
}

func TestHandleGetHostInventory(t *testing.T) {
	w := makeRequest("GET", `/ansible/hosts/test`, nil)
	if w.Code != 200 {
		t.Fail()
	}

	j, err := simplejson.NewFromReader(w.Body)
	if err != nil {
		t.Fail()
	}

	if _, ok := j.CheckGet("_meta"); !ok {
		t.Fail()
	}
}
