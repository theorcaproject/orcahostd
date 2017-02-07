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


package model

type Application struct {
	Name     string
	State    string
	Version  string
	ChangeId string
	Metrics  Metric
}

type ApplicationState struct {
	DockerAppId string
	Name        string
	Application Application
}

type HostCheckinDataPackage struct {
	State          []*ApplicationState
	ChangesApplied map[string]bool
	HostMetrics    Metric
}

type Change struct {
	Id     string
	Type   string
	Name 	string
	Version string

	AppConfig VersionConfig
}

type DockerConfig struct {
	Tag        string
	Repository string
	Reference  string
}

type PortMapping struct {
	HostPort      string
	ContainerPort string
}

type VolumeMapping struct {
	HostPath      string
	ContainerPath string
}

type File struct {
	HostPath           string
	Base64FileContents string
}

type EnvironmentVariable struct {
	Key   string
	Value string
}

type VersionConfig struct {
	DockerConfig	     DockerConfig
	PortMappings         []PortMapping
	VolumeMappings       []VolumeMapping
	EnvironmentVariables []EnvironmentVariable
	Files                []File
	Version 	     string
}

type Metric struct {
	CpuUsage int64
	MemoryUsage int64
	NetworkUsage int64
}
