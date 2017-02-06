package model

type Application struct {
	Name     string
	State    string
	Version  string
	ChangeId string
	Metrics  string
}

type ApplicationState struct {
	DockerAppId string
	Name        string
	Application Application
}

type HostCheckinDataPackage struct {
	State          []*ApplicationState
	ChangesApplied map[string]bool
	Metrics        map[string]Metric
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
	CpuUsage uint64
	MemoryUsage uint64
	NetworkUsage uint64
}
