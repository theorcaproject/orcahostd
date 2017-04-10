package logreceiver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"orcahostd/client"
	Logger "orcahostd/logs"
	"strings"
)

var ElkLogReceiverLogger = Logger.LoggerWithField(Logger.Logger, "module", "elklogreceiver")

type ElkLogSender struct {
	uri    string
	hostId string
	user   string
	passwd string
}

func (logSender *ElkLogSender) Init(uri string, hostId string, user string, passwd string) {
	logSender.uri = uri
	logSender.hostId = hostId
	logSender.user = user
	logSender.passwd = passwd
}

func (logSender *ElkLogSender) postLogs(app string, message string, logLevel string) {
	client := &http.Client{}
	b := new(bytes.Buffer)
	jsonErr := json.NewEncoder(b).Encode(map[string]interface{}{"app": app, "message": message, "logLevel": logLevel})
	if jsonErr != nil {
		ElkLogReceiverLogger.Errorf("Could not encode Logs: %+v.", jsonErr)
		return
	}
	req, _ := http.NewRequest("PUT", logSender.uri, b)
	req.Header.Set("Content-Type", "application/json")
	// req.SetBasicAuth(logSender.user, logSender.passwd)
	res, err := client.Do(req)
	if err != nil {
		ElkLogReceiverLogger.Errorf("Could not send logs to ELK: %+v", err)
	} else {
		defer res.Body.Close()
	}
}

func (logSender *ElkLogSender) PushLogs(logs map[string]client.Logs) {
	for app, appLogs := range logs {
		if len(appLogs.StdErr) > 0 {
			entries := strings.Split(appLogs.StdErr, "\n")
			for i := len(entries) - 1; i >= 0; i-- {
				logSender.postLogs(app, entries[i], "stderr")
			}
		}

		if len(appLogs.StdOut) > 0 {
			entries := strings.Split(appLogs.StdOut, "\n")
			for i := len(entries) - 1; i >= 0; i-- {
				logSender.postLogs(app, entries[i], "stdout")
			}
		}
	}
}
