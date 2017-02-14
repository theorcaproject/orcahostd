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


package client

import (
	Logger "orcahostd/logs"
	"orcahostd/docker"
	"fmt"
	"math/rand"
	"orcahostd/model"
	"errors"
	"time"
	"net/http"
	"net"
)

var ClientLogger = Logger.LoggerWithField(Logger.Logger, "module", "client")
var cli Client

type Client struct {
	AppState []*model.ApplicationState
	AppConfiguration map[string]model.VersionConfig
	Changes map[string]bool

	engine docker.DockerContainerEngine
}

type Logs struct {
	StdOut string
	StdErr string
}

func (client *Client) Init() {
	ClientLogger.Info("Initializing Client...")
	client.AppState = make([]*model.ApplicationState, 0)
	client.Changes = make(map[string]bool)

	client.engine = docker.DockerContainerEngine{}
	client.engine.Init()
}

func (client *Client) HandleRequestedChanges(changes []model.Change) {
	for _, change := range changes {
		/* First check that we have not already dealth with this change */
		if _, ok := client.Changes[change.Id]; ok {
			continue
		}

		if change.Type == "add_application" {
			/* First things first, check that we do not already have this application. If we do, nuke it */
			_, err := client.GetAppStateIndividual(change.Name)
			if err == nil {
				client.DeleteApp(change.Name)
			}

			if client.DeployApp(change.Name, change.AppConfig) {
				client.Changes[change.Id] = true
			}
		}

		if change.Type == "remove_application" {
			if client.DeleteApp(change.Name) {
				client.Changes[change.Id] = true
			}
		}
	}
}

func GenerateId(app string) string {
	return string(fmt.Sprintf("%s_%d", app, rand.Int31()))
}

func (client *Client) RunCheck(config model.VersionConfig) bool {
	for _, change := range config.Checks {
		if change.Type == "http" {
			res, err := http.Get(change.Goal)
			if err != nil || res.StatusCode != 200{
				return false
			}

		}else if change.Type == "tcp"{
			_, err := net.Dial("tcp", change.Goal)
			if err != nil {
				return false
			}
		}
	}

	return true
}

func (client *Client) DeployApp(name string, config model.VersionConfig) bool {
	ClientLogger.Infof("Installing app %s:%d", name, config.Version)
	client.engine.InstallApp(name, config)

	ClientLogger.Infof("Starting app %s:%d", name, config.Version)
	id := GenerateId(name)
	newAppState := &model.ApplicationState{
		Name: name,
		DockerAppId: id,
		Application: model.Application{
			State:"",
			ChangeId:"",
			Name:name,
			Version: config.Version,
		},
	}

	client.AppState = append(client.AppState, newAppState)
	/* Add the configuration for this application */
	client.AppConfiguration[name] = config
	res := client.engine.RunApp(id, name, config)
	if !res {
		newAppState.Application.State = "installation_failed"
	}else{
		for i := 1; i <= 10; i++ {
			if !client.RunCheck(config) {
				if i == 10 {
					newAppState.Application.State = "checks_failed"
					break
				}

				time.Sleep(time.Duration(6 * time.Second))
				continue
			} else {
				newAppState.Application.State = "running"
				break
			}
		}
	}

	ClientLogger.Infof("Starting app %s:%d done. Success=%t", name, config.Version, res)
	return res
}


func (client *Client) DeleteApp(name string) bool {
	ClientLogger.Infof("Starting deletion of app %s", name)
	app, err := client.GetAppStateIndividual(name)
	if err == nil {
		client.engine.StopApp(app.DockerAppId)
		client.DelAppStateIndividual(name)
	}

	return true;
}

func (client *Client) GetAppMetrics() map[string]model.Metric {
	ret := make(map[string]model.Metric)
	for _, application := range client.AppState {
		metric, _ := client.engine.AppMetrics(application.DockerAppId)
		ret[application.Name] = metric
	}
	return ret
}

func (client *Client) GetAppLogs() map[string]Logs {
	ret := make(map[string]Logs)
	for _, application := range client.AppState {
		out, err := client.engine.AppLogs(application.DockerAppId)
		ret[application.Name] = Logs{ StdOut: out, StdErr:err}
	}
	return ret
}

func (client *Client) GetHostMetrics() model.Metric{
	return client.engine.HostMetrics()
}

func (client *Client) GetAppState() []*model.ApplicationState{
	// We need to update the AppState before returning it:
	for _, state := range client.AppState {
		if client.engine.QueryApp(state.DockerAppId) {
			appConfiguration := client.AppConfiguration[state.Name]

			if !client.RunCheck(appConfiguration) {
				state.Application.State = "checks_failed"
			}else{
				state.Application.State = "running"
			}
		}else{
			state.Application.State = "failed"
		}
	}

	return client.AppState
}

func (client *Client) GetAppStateIndividual(application string) (*model.ApplicationState, error){
	// We need to update the AppState before returning it:
	for _, state := range client.AppState {
		if state.Name == application {
			return state, nil
		}
	}

	return nil, errors.New("No application")
}

func (client *Client) DelAppStateIndividual(application string){
	// We need to update the AppState before returning it:
	states := make([]*model.ApplicationState, 0)

	for _, state := range client.AppState {
		if state.Name != application {
			states = append(states, state)
		}
	}

	client.AppState = states
}

func (client *Client) GetChangeLog() map[string]bool {
	return client.Changes
}