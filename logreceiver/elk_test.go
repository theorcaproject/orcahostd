package logreceiver_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"orcahostd/client"
	"orcahostd/logreceiver"
	"testing"
)

func TestSendStdOut(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		type Data struct {
			App      string
			LogLevel string
			Message  string
		}
		var d Data
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &d)
		if d.App != "my_app" || d.LogLevel != "stdout" || d.Message != "stdout log message" {
			t.Errorf("EXPECTED (app:%q logLevel:%q message:%q) GOT (app:%q logLevel:%q message:%q)",
				"my_app", "stdout", "stdout log messag", d.App, d.LogLevel, d.Message)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Done"))
	}))
	defer srv.Close()
	defer func() {
		if !called {
			t.Errorf("Route for SendStdOut wasn't called")
		}
	}()

	sender := new(logreceiver.ElkLogSender)
	sender.Init(srv.URL, "hostId", "user", "passwd")
	logs := make(map[string]client.Logs)
	logs["my_app"] = client.Logs{StdOut: "stdout log message", StdErr: ""}
	sender.Send(logs)
}

func TestSendStdErr(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		type Data struct {
			App      string
			LogLevel string
			Message  string
		}
		var d Data
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &d)
		if d.App != "my_app" || d.LogLevel != "stderr" || d.Message != "stderr log message" {
			t.Errorf("EXPECTED (app:%q logLevel:%q message:%q) GOT (app:%q logLevel:%q message:%q)",
				"my_app", "stderr", "stderr log messag", d.App, d.LogLevel, d.Message)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Done"))
	}))
	defer srv.Close()
	defer func() {
		if !called {
			t.Errorf("Route for SendStdErr wasn't called")
		}
	}()

	sender := new(logreceiver.ElkLogSender)
	sender.Init(srv.URL, "hostId", "user", "passwd")
	logs := make(map[string]client.Logs)
	logs["my_app"] = client.Logs{StdOut: "", StdErr: "stderr log message"}
	sender.Send(logs)
}
