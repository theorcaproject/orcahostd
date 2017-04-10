package logreceiver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"orcahostd/client"
	Logger "orcahostd/logs"
)

var TrainerLogReceiverLogger = Logger.LoggerWithField(Logger.Logger, "module", "trainerlogreceiver")

type TrainerLogSender struct {
	uri    string
	hostId string
}

func (logSender *TrainerLogSender) Init(uri string, hostId string) {
	logSender.uri = uri
	logSender.hostId = hostId
}

func (logSender *TrainerLogSender) PushLogs(logs map[string]client.Logs) {
	b := new(bytes.Buffer)
	jsonErr := json.NewEncoder(b).Encode(logs)
	if jsonErr != nil {
		TrainerLogReceiverLogger.Errorf("Could not encode Logs: %+v.", jsonErr)
		return
	}
	res, err := http.Post(logSender.uri+"/log/apps?host="+logSender.hostId, "application/json; charset=utf-8", b)
	if err != nil {
		TrainerLogReceiverLogger.Errorf("Could not send logs to trainer: %+v", err)
	} else {
		defer res.Body.Close()
	}
}
