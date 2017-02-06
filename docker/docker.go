/*
Copyright Alex Mack
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
	Logger "gatoor/orca/util/log"
	DockerClient "github.com/fsouza/go-dockerclient"
	"bytes"
	"fmt"
	"bluewhale/orcahostd/model"
)

var DockerLogger = Logger.LoggerWithField(Logger.Logger, "module", "docker")
var dockerCli *DockerClient.Client

type DockerContainerEngine struct {

}

func (c *DockerContainerEngine) Init() {
	var err error
	dockerCli, err = DockerClient.NewClient("unix:///var/run/docker.sock")

	if err != nil {
		DockerLogger.Fatalf("Docker client could not be instantiated: %v", err)
	}
}

func DockerCli() *DockerClient.Client {
	if dockerCli == nil {
		DockerLogger.Infof("DockerClient was nil, instantiating again.")
		var err error
		dockerCli, err = DockerClient.NewClient("unix:///var/run/docker.sock")

		if err != nil {
			DockerLogger.Fatalf("Docker client could not be instantiated: %v", err)
		}
	}
	return dockerCli
}

func (c *DockerContainerEngine) InstallApp(name string, config model.VersionConfig) bool {
	DockerLogger.Infof("Installing docker app %s", name)
	var buf bytes.Buffer
	imageOpt := DockerClient.PullImageOptions{
		Repository: config.DockerConfig.Repository,
		Tag: config.DockerConfig.Tag,
		OutputStream: &buf,
	}
	err := DockerCli().PullImage(imageOpt, DockerClient.AuthConfiguration{})
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

	hostConfig := DockerClient.HostConfig{PortBindings: bindings, PublishAllPorts:true}
	config := DockerClient.Config{AttachStdout: true, AttachStdin: true, Image: fmt.Sprintf("%s:%s", appConf.DockerConfig.Repository, appConf.DockerConfig.Tag), ExposedPorts:ports}
	opts := DockerClient.CreateContainerOptions{Name: string(appId), Config: &config, HostConfig:&hostConfig}
	container, containerErr := DockerCli().CreateContainer(opts)
	if containerErr != nil {
		DockerLogger.Errorf("Running docker app %s with error %s", appId, containerErr)
		return false
	}

	err := DockerCli().StartContainer(container.ID, &hostConfig)
	if err != nil {
		DockerLogger.Errorf("Running docker app %s with error %s", appId, err)
		return false
	}
	DockerLogger.Infof("Running docker app %s - %s successful", appId)
	return true
}


func (c *DockerContainerEngine) QueryApp(appId string) bool {
	DockerLogger.Debugf("Query docker app %s", appId)
	resp, err := DockerCli().InspectContainer(string(appId))
	if err != nil {
		DockerLogger.Debugf("Query docker app %s failed: %s", appId, err)
		return false
	}
	DockerLogger.Debugf("Query docker app %s - successful %+v", appId, resp)
	return resp.State.Running
}

func (c *DockerContainerEngine) StopApp(appId string) bool {
	DockerLogger.Infof("Stopping docker app %s", appId)
	err := DockerCli().StopContainer(fmt.Sprintf("%s", appId), 0)
	fail := false
	if err != nil {
		DockerLogger.Infof("Stopping docker app %s - failed: %s", appId, err)
		fail = true
	}
	opts := DockerClient.RemoveContainerOptions{ID: string(appId)}
	err = DockerCli().RemoveContainer(opts)
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

func (c *DockerContainerEngine) AppMetrics(appId string) model.Metric {
	DockerLogger.Debugf("Getting AppMetrics for app %s %s:%d", appId)
	errC := make(chan error, 1)
	statsC := make(chan *DockerClient.Stats)
	done := make(chan bool)

	go func() {
		errC <- DockerCli().Stats(DockerClient.StatsOptions{ID: string(appId), Stats: statsC, Stream: true, Done: done})
		close(errC)
	}()
	var resultStats []*DockerClient.Stats
	count := 0
	for {
		count++
		stats, ok := <-statsC
		if !ok || count > 2 {
			close(done)
			break
		}
		resultStats = append(resultStats, stats)
	}
	//err := <-errC
	//if (err != nil){
	//	DockerLogger.Infof("Getting AppMetrics for app %s %s:%d failed: %s. Only %d results", appId, appConf.Name, appConf.Version, err, len(resultStats))
	//	return false
	//}
	return parseDockerStats(resultStats[0], resultStats[1])
}

func parseDockerStats(stat0 *DockerClient.Stats, stat1 *DockerClient.Stats) model.Metric {
	var (
		cpuPercent = uint64(0)
		cpuDelta = float64(stat1.CPUStats.CPUUsage.TotalUsage) - float64(stat0.CPUStats.CPUUsage.TotalUsage)
		systemDelta = float64(stat1.CPUStats.SystemCPUUsage) - float64(stat0.CPUStats.SystemCPUUsage)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = uint64((cpuDelta / systemDelta) * float64(len(stat1.CPUStats.CPUUsage.PercpuUsage)) * 100.0)
	}

	metric := model.Metric{}
	metric.CpuUsage = cpuPercent
	metric.MemoryUsage = (stat1.MemoryStats.Usage + stat0.MemoryStats.Usage) / 2
	metric.NetworkUsage = (stat1.Network.RxBytes + stat0.Network.RxBytes) / 2
	return metric
}
