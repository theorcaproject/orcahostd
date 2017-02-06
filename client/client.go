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
)

var ClientLogger = Logger.LoggerWithField(Logger.Logger, "module", "client")
var cli Client

type Client struct {
	AppState []*model.ApplicationState
	Changes map[string]bool

	engine docker.DockerContainerEngine
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
	res := client.engine.RunApp(id, name, config)
	if !res {
		newAppState.Application.State = "failed"
	}else{
		newAppState.Application.State = "running"
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
		ret[application.Name] = client.engine.AppMetrics(application.DockerAppId)
	}
	return ret
}

func (client *Client) GetAppState() []*model.ApplicationState{
	// We need to update the AppState before returning it:
	for _, state := range client.AppState {
		if client.engine.QueryApp(state.DockerAppId) {
			state.Application.State = "running"
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