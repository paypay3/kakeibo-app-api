package testutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func SetUpMockServer() func() {
	if err := os.Setenv("USER_HOST", "localhost"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	verifyGroupAffiliationHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ts := httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: verifyGroupAffiliationHandler},
	}
	ts.Start()

	return func() {
		ts.Close()
	}
}

func GetRequestJsonFromTestData(t *testing.T) string {
	t.Helper()

	requestFilePath := filepath.Join("testdata", t.Name(), "request.json")

	byteData, err := ioutil.ReadFile(requestFilePath)
	if err != nil {
		t.Fatalf("unexpected error while opening file '%#v'", err)
	}

	return string(byteData)
}

func AssertResponseHeader(t *testing.T, res *http.Response, wantStatusCode int) {
	t.Helper()

	if diff := cmp.Diff(wantStatusCode, res.StatusCode); len(diff) != 0 {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}

	if diff := cmp.Diff("application/json; charset=UTF-8", res.Header.Get("Content-Type")); len(diff) != 0 {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}
}

func AssertResponseBody(t *testing.T, res *http.Response, wantStruct interface{}, gotStruct interface{}) {
	t.Helper()

	goldenFilePath := filepath.Join("testdata", t.Name(), "response.json.golden")

	wantData, err := ioutil.ReadFile(goldenFilePath)
	if err != nil {
		t.Fatalf("unexpected error by ioutil.ReadFile '%#v'", err)
	}

	gotData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("unexpected error by ioutil.ReadAll() '%#v'", err)
	}

	if err := json.Unmarshal(wantData, wantStruct); err != nil {
		t.Fatalf("unexpected error by json.Unmarshal() '%#v'", err)
	}

	if err := json.Unmarshal(gotData, gotStruct); err != nil {
		t.Fatalf("unexpected error by json.Unmarshal() '%#v'", err)
	}

	if diff := cmp.Diff(wantStruct, gotStruct); len(diff) != 0 {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}
}
