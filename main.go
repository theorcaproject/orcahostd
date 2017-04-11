/*
Copyright Alex Mack and Michael Lawson (michael@sphinix.com)
This file is part of Orca.

Orca is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Orca is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Orca.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"orcahostd/client"
	"orcahostd/logreceiver"
	Logger "orcahostd/logs"
	"orcahostd/model"
	"time"
)

var MainLogger = Logger.LoggerWithField(Logger.Logger, "module", "main")

func main() {
	var hostId = flag.String("hostid", "host1", "Host Identifier")
	var checkInInterval = flag.Int("interval", 60, "Check in interval")
	var trainerUri = flag.String("traineruri", "http://localhost:5001", "Trainer Uri")
	var elkUri = flag.String("elkuri", "", "ELK logstash Uri")
	var elkUser = flag.String("elkuser", "", "ELK user name")
	var elkPasswd = flag.String("elkpasswd", "", "ELK password")
	flag.Parse()
	client := client.Client{}
	client.Init()

	logSenders := make([]logreceiver.LogSender, 0)

	trainerLogSender := new(logreceiver.TrainerLogSender)
	trainerLogSender.Init((*trainerUri), (*hostId))
	logSenders = append(logSenders, trainerLogSender)

	if len((*elkUri)) > 0 && len((*elkUser)) > 0 && len((*elkPasswd)) > 0 {
		elkLogSender := new(logreceiver.ElkLogSender)
		elkLogSender.Init((*elkUri), (*hostId), (*elkUser), (*elkPasswd))
		logSenders = append(logSenders, elkLogSender)
	}

	logsTicker := time.NewTicker(time.Duration(10 * time.Second))
	go func() {
		for {
			<-logsTicker.C
			logs := client.GetAppLogs()
			for _, logSender := range logSenders {
				logSender.Send(logs)
			}
		}
	}()
	trainerTicker := time.NewTicker(time.Duration((*checkInInterval)) * time.Second)
	func() {
		for {
			<-trainerTicker.C
			CallTrainer((*trainerUri), (*hostId), &client)
		}
	}()
}

func CallTrainer(trainerUri string, hostId string, client *client.Client) {
	MainLogger.Infof("Calling Trainer...")
	metrics := client.GetAppMetrics()
	hostMetrics := client.GetHostMetrics()
	state := client.GetAppState()

	for _, object := range state {
		object.Application.Metrics = metrics[object.Name]
	}

	dataPackage := model.HostCheckinDataPackage{
		State:          state,
		ChangesApplied: client.GetChangeLog(),
		HostMetrics:    hostMetrics,
	}

	b := new(bytes.Buffer)
	jsonErr := json.NewEncoder(b).Encode(dataPackage)
	if jsonErr != nil {
		MainLogger.Errorf("Could not encode Metrics: %+v. Sending without metrics.", jsonErr)
		return
	}

	res, err := http.Post(trainerUri+"/checkin?host="+hostId, "application/json; charset=utf-8", b)
	if err != nil {
		MainLogger.Errorf("Could not send data to trainer: %+v", err)
	} else {
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			MainLogger.Errorf("Could not read reponse from trainer: %+v", err)
		} else {
			var changes = make([]model.Change, 0)
			if err := json.Unmarshal(body, &changes); err != nil {
				MainLogger.Errorf("Failed to parse response - %s HTTP_BODY: %s", err, string(body))
			} else {

				/* This seems a bit funny, but since we are only dealing with one change at a time this
				speeds things up. */
				if client.HandleRequestedChanges(changes) {
					CallTrainer(trainerUri, hostId, client)
				}
			}
		}
	}
}
