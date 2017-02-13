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


package docker

import (
	Logger "orcahostd/logs"
	DockerClient "github.com/fsouza/go-dockerclient"
	"bytes"
	"fmt"
	"orcahostd/model"
	"os"
	"errors"
)


var DockerLogger = Logger.LoggerWithField(Logger.Logger, "module", "docker")

type DockerContainerEngine struct {
	dockerCli *DockerClient.Client

	metrics map[string]DockerMetrics
}

func (c *DockerContainerEngine) Init() {
	var err error
	c.dockerCli, err = DockerClient.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		DockerLogger.Fatalf("Docker client could not be instantiated: %v", err)
	}
}

//func DockerCli() *DockerClient.Client {
//	if c.dockerCli == nil {
//		DockerLogger.Infof("DockerClient was nil, instantiating again.")
//		var err error
//		c.dockerCli, err = DockerClient.NewClient("unix:///var/run/docker.sock")
//
//		if err != nil {
//			DockerLogger.Fatalf("Docker client could not be instantiated: %v", err)
//		}
//	}
//	return dockerCli
//}

func (c *DockerContainerEngine) InstallApp(name string, config model.VersionConfig) bool {
	DockerLogger.Infof("Installing docker app %s", name)
	var buf bytes.Buffer
	authOpt := DockerClient.AuthConfiguration{
		Username: config.DockerConfig.Username,
		Password: config.DockerConfig.Password,
		Email: config.DockerConfig.Email,
		ServerAddress: config.DockerConfig.Server,
	}
	imageOpt := DockerClient.PullImageOptions{
		Repository: config.DockerConfig.Repository,
		Tag: config.DockerConfig.Tag,
		OutputStream: &buf,
	}
	err := c.dockerCli.PullImage(imageOpt, authOpt)
	if err != nil {
		DockerLogger.Errorf("Install of app %s failed: %s", name, err)
		return false
	}

	DockerLogger.Infof("Install of app %s successful", name)
	return true
}


func (c *DockerContainerEngine) RunApp(appId string, name string, appConf model.VersionConfig) bool {
	bindings := make(map[DockerClient.Port][]DockerClient.PortBinding)
	ports := make(map[DockerClient.Port]struct{})
	for _, v := range appConf.PortMappings {
		bindings[DockerClient.Port(v.ContainerPort)] = []DockerClient.PortBinding{DockerClient.PortBinding{HostPort: v.HostPort}}
		ports[DockerClient.Port(v.ContainerPort)] = struct{}{}
	}
	DockerLogger.Warnf("Bindinds are %+v", bindings)

	env := DockerClient.Env{}
	for _, item := range appConf.EnvironmentVariables{
		env.Set(item.Key, item.Value)
	}

	/* Handle Files */
	os.Mkdir("/tmp/" + appId, 600)
	for _, file := range appConf.Files {
		fp, err := os.Create("/tmp/" + appId + file.HostPath)
		if err == nil {
			fp.WriteString(file.Base64FileContents)
			fp.Close()
		}
	}

	mounts := make([]string, 1)
	mounts[0] = "/tmp/" + appId + ":/orcatmp"

	hostConfig := DockerClient.HostConfig{PortBindings: bindings, PublishAllPorts:true, Binds:mounts}
	config := DockerClient.Config{AttachStdout: true, AttachStdin: true, Image: fmt.Sprintf("%s:%s", appConf.DockerConfig.Repository, appConf.DockerConfig.Tag), ExposedPorts:ports, Env:env,}
	opts := DockerClient.CreateContainerOptions{Name: string(appId), Config: &config, HostConfig:&hostConfig}
	container, containerErr :=c.dockerCli.CreateContainer(opts)
	if containerErr != nil {
		DockerLogger.Errorf("Running docker app %s with error %s", appId, containerErr)
		return false
	}

	err := c.dockerCli.StartContainer(container.ID, &hostConfig)
	if err != nil {
		DockerLogger.Errorf("Running docker app %s with error %s", appId, err)
		return false
	}
	DockerLogger.Infof("Running docker app %s - %s successful", appId)
	return true
}


func (c *DockerContainerEngine) QueryApp(appId string) bool {
	DockerLogger.Debugf("Query docker app %s", appId)
	resp, err := c.dockerCli.InspectContainer(string(appId))
	if err != nil {
		DockerLogger.Debugf("Query docker app %s failed: %s", appId, err)
		return false
	}
	DockerLogger.Debugf("Query docker app %s - successful %+v", appId, resp)
	return resp.State.Running
}

func (c *DockerContainerEngine) StopApp(appId string) bool {
	DockerLogger.Infof("Stopping docker app %s", appId)
	err := c.dockerCli.StopContainer(fmt.Sprintf("%s", appId), 0)
	fail := false
	if err != nil {
		DockerLogger.Infof("Stopping docker app %s - failed: %s", appId, err)
		fail = true
	}
	opts := DockerClient.RemoveContainerOptions{ID: string(appId)}
	err = c.dockerCli.RemoveContainer(opts)
	if err != nil {
		DockerLogger.Infof("Stopping docker app %s - %s", appId, err)
		fail = true
	}
	if fail {
		return false
	}
	DockerLogger.Infof("Stopping docker app %s - successful", appId)
	return true
}

type DockerMetrics struct {
	errC chan error
	statsC chan *DockerClient.Stats
	done chan bool
}


func (c *DockerContainerEngine) AppMetrics(appId string) (model.Metric, error) {
	DockerLogger.Debugf("Getting AppMetrics for app %s %s:%d", appId)

	if _, ok := c.metrics[appId] !ok {
		metricsItem := &DockerMetrics{
			done: make (chan bool),
			errC: make (chan error),
			statsC: make (chan *DockerClient.Stats),
		}
		c.metrics[appId] = metricsItem

		DockerLogger.Debugf("Creating DockerMetrics Entity for app %s", appId)
		go func() {
			c.dockerCli.Stats(DockerClient.StatsOptions{ID: string(appId), Stats: metricsItem.statsC, Stream: true, Done: metricsItem.done})
			close(metricsItem.errC)
		}()
	}

	entry := c.metrics[appId]

	var resultStats []*DockerClient.Stats
	count := 0
	for {
		count++
		stats, ok := <-entry.statsC
		if !ok || count > 2 {
			break
		}
		resultStats = append(resultStats, stats)
	}

	if len(resultStats) != 2 {
		return model.Metric{}, errors.New("Could not collect metrics, there were no stats")
	}

	return parseDockerStats(resultStats[0], resultStats[1])
}


func parseDockerStats(stat0 *DockerClient.Stats, stat1 *DockerClient.Stats) (model.Metric, error) {
	if stat0 == nil || stat1 == nil {
		return model.Metric{}, errors.New("Could not collect metrics")
	}

	var (
		cpuPercent = uint64(0)
		cpuDelta = float64(stat1.CPUStats.CPUUsage.TotalUsage) - float64(stat0.CPUStats.CPUUsage.TotalUsage)
		systemDelta = float64(stat1.CPUStats.SystemCPUUsage) - float64(stat0.CPUStats.SystemCPUUsage)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = uint64((cpuDelta / systemDelta) * float64(len(stat1.CPUStats.CPUUsage.PercpuUsage)) * 100.0)
	}

	metric := model.Metric{}
	metric.CpuUsage = int64(cpuPercent)
	metric.MemoryUsage = int64((stat1.MemoryStats.Usage + stat0.MemoryStats.Usage) / 2)
	metric.NetworkUsage = int64((stat1.Network.RxBytes + stat0.Network.RxBytes) / 2)
	return metric, nil
}
