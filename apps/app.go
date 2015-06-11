package apps

import (
	"encoding/json"
	"github.com/CiscoCloud/marathon-consul/utils"
	"github.com/hashicorp/consul/api"
)

type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	ServicePort   int    `json:"servicePort"`
	Protocol      string `json:"protocol"`
}

type Parameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Docker struct {
	Image          string        `json:"image"`
	Parameters     []Parameter   `json:"parameters"`
	Privileged     bool          `json:"privileged"`
	Network        string        `json:"network"`
	PortMappings   []PortMapping `json:"portMappings"`
	ForcePullImage bool          `json:"forcePullImage"`
}

type Volume struct {
	ContainerPath string `json:"containerPath"`
	HostPath      string `json:"hostPath"`
	Mode          string `json:"mode"`
}

type Container struct {
	Docker  *Docker  `json:"docker"`
	Type    string   `json:"type"`
	Volumes []Volume `json:"volumes"`
}

type HealthCheck struct {
	Path                   string `json:"path"`
	PortIndex              int    `json:"portIndex"`
	Protocol               string `json:"protocol"`
	GracePeriodSeconds     int    `json:"gracePeriodSeconds"`
	IntervalSeconds        int    `json:"intervalSeconds"`
	TimeoutSeconds         int    `json:"timeoutSeconds"`
	MaxConsecutiveFailures int    `json:"maxConsecutiveFailures"`
}

type UpgradeStrategy struct {
	MinimumHealthCapacity float64 `json:"minimumHealthCapacity"`
	MaximumOverCapacity   float64 `json:"maximumOverCapacity"`
}

type App struct {
	Args            []string          `json:"args"`
	BackoffFactor   float64           `json:"backoffFactor"`
	BackoffSeconds  int               `json:"backoffSeconds"`
	Cmd             string            `json:"cmd"`
	Constraints     [][]string        `json:"constraints"`
	Container       *Container        `json:"container"`
	CPUs            float64           `json:"cpus"`
	Dependencies    []string          `json:"dependencies"`
	Disk            float64           `json:"disk"`
	Env             map[string]string `json:"env"`
	Executor        string            `json:"executor"`
	Labels          map[string]string `json:"labels"`
	HealthChecks    []HealthCheck     `json:"healthChecks"`
	ID              string            `json:"id"`
	Instances       int               `json:"instances"`
	Mem             float64           `json:"mem"`
	Ports           []int             `json:"ports"`
	RequirePorts    bool              `json:"requirePorts"`
	StoreUrls       []string          `json:"storeUrls"`
	UpgradeStrategy UpgradeStrategy   `json:"upgradeStrategy"`
	Uris            []string          `json:"uris"`
	User            string            `json:"user"`
	Version         string            `json:"version"`
}

func (app *App) KV() *api.KVPair {
	serialized, _ := json.Marshal(app)

	return &api.KVPair{
		Key:   app.Key(),
		Value: serialized,
	}
}

func (app *App) Key() string {
	return utils.CleanID(app.ID)
}
