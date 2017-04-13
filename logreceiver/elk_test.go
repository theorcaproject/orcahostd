package logreceiver_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"orcahostd/client"
	"orcahostd/logreceiver"
	"os"
	"path/filepath"
	"testing"
)

const pemPublicKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAzeD/7lZL5gfyB2hCSSVHUPuYn++E7YhK4v7vpNEgTl8rZwQL
btf0LrwPlQBQpI5USWsAUed8KuZPMVsmZ/YZ/I9qqlpKTXui99dS2rbHb/KKDsL5
yEJ/zTI7Ik0Jnq9ETQOXnaPx8UnnFXtmkGadylVzUKSexJVll2Sap98Klqcsejoa
4l5Y7UIHC9mWQnN5ZjbKkKtHNYT/tAz5bDVA3XSwqvNEIqUQ8qMBaSXMDzTNIkve
3pGDWGQyPZGDsanXSGkS4KYKco+Hso/xWK9kwR7e5T7/jiOngTXECgqz4NYrsa8w
05bjxsegrAYwjZ1suUf9JGSql3XoGwQ0l7OgCQIDAQABAoIBADz5kJM0N9JvM/2B
oXAsfEy911w3AjWfkh5cxvkXfuv3P9GI3yH7D7TvueW1bCzwgoIkELoxRxMllvrV
NjDAML4ec8b0auE75u5kdYOVcsfzG3R3xqbLGzLY1663BkxbOG1ezP7BWZzO+IPi
QmQuIDmWyUpvFx696JLIFw30/xvS61YejBtmX6FKhmbLl4dcuutSaVU0ZVQesslt
cOC6C8av74vmN6YUMWRPTV4+iSdmLQEDiwv1Vra+ZhzLlbEbQZ97SnHP4Kl355fF
hOly9bphe77ycE+T98TiwS3c35dZj3BFkDnX2FGtg2L1AxpmJXwGUjTQX3opsEkc
ZWOBT0kCgYEA6TFRs03EHUY31VA4bSAlxNZOFFp9eI73bWYYdHkrcQX35F8ryzkF
FqAJz4PNYZbsJ3noGCIhtZdbBM9ow0yZEy2U8oP+TBJz8X6AyhZcIeu/LIl8f7n0
8Trzf4BiwS2hb90Pad0Q0suIz30+Arw8JOPBFlCNeqtji74qD0t8BcMCgYEA4gPM
cNcELjQ9c3nr8EAHPty0ZS7ZcV7iGf35PKvHW9IptCmEX0ul8j6Kt7EeLk1ZVtL7
trRpphSIz2BeDizDEp9yKK7RpvyTVF4k4UPbW1j1XVzCe5Tsia8hqdkbOlBTUmz4
nL1IO9orXMUNlECFZNJI9nRsU1CH3puEg/8hikMCgYEAi05yP0pSwSZEjoM44kAV
MAzSYihY0l+eAlW+gD4urHtjRqNwNxxeJNEAa16SoB0YANE7zMb/GktMDYiWTi2B
OMq/M02U6f8QEpF/ALrw2TbLYyDTJj6BzGZqNp4M4NiQm5IU9iohNbxvg3yPQfUP
fP4uSFVg34ppkn7NA4wVkB8CgYBx7S5BdvDhhW2wZrW6fdvpIQFBu7LZxdU8+tuG
bKRqMW7aJM9X5d75U/NCkuI+vriY3nMJbrmOgO1RcycWCBQwr/Swcya1AL4XGfmH
H1hUHGxaKmbSOohdAs16OzjRVSoa81kCURs3KEsRUTb+EuPqpWEn8hmkiYRjfor2
qkUy1wKBgAPqO+ixggr8DhjsFt72LSTmMi+Bo5hgnyM3U2Zg4gnigqLMHQSB1BG9
oDOBvFvZkFK1R6LBf3wkprK+yR0XCC7np/qJek88xwy4xM+3vQXAZ50vX3GTAgob
yJvhGUmdmt39/1CWegYtr+7HFzSAjmzvsOw1LzK/Qoyu0b4O2nHs
-----END RSA PRIVATE KEY-----`

func setUp() (tmpDir string, certPath string) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Println(err)
	}
	certPath = tmpDir + string(filepath.Separator) + "cert.pem"
	ioutil.WriteFile(certPath, []byte(pemPublicKey), 0644)
	return tmpDir, certPath
}

func tearDown(dir string) {
	os.RemoveAll(dir)
}

func TestSendStdOut(t *testing.T) {
	dir, certPath := setUp()
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
	defer tearDown(dir)

	sender := new(logreceiver.ElkLogSender)
	sender.Init(srv.URL, "hostId", certPath)
	logs := make(map[string]client.Logs)
	logs["my_app"] = client.Logs{StdOut: "stdout log message", StdErr: ""}
	sender.Send(logs)
}

func TestSendStdErr(t *testing.T) {
	dir, certPath := setUp()
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
	defer tearDown(dir)

	sender := new(logreceiver.ElkLogSender)
	sender.Init(srv.URL, "hostId", certPath)
	logs := make(map[string]client.Logs)
	logs["my_app"] = client.Logs{StdOut: "", StdErr: "stderr log message"}
	sender.Send(logs)
}
