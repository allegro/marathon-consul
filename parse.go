package main

import (
	"encoding/json"
	"errors"
	"strings"
)

var ErrNoApps = errors.New("no apps present in provided JSON")

type Docker struct {
	Image      string   `json:"image"`
	Parameters []string `json:"parameters"`
	Privileged bool     `json:"privileged"`
}

type Volume struct {
	ContainerPath string `json:"containerPath"`
	HostPath      string `json:"hostPath"`
	Mode          string `json:"mode"`
}

type Container struct {
	Docker  *Docker  `json:"docker"`
	Type    bool     `json:"type"`
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
	Constraints     []string          `json:"constraints"`
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

type APIPostEvent struct {
	Type string `json:"eventType"`
	App  App    `json:"appDefinition"`
}

type DeploymentInfoEvent struct {
	Type string `json:"eventType"`
	Plan struct {
		Target struct {
			Apps []*App `json:"apps"`
		} `json:"target"`
	} `json:"plan"`
}

func ParseApps(event []byte) (apps []*App, err error) {
	if strings.Index(string(event), "api_post_event") != -1 {
		container := APIPostEvent{}
		err = json.Unmarshal(event, &container)
		if err != nil {
			return nil, err
		}

		apps = []*App{&container.App}
	} else if strings.Index(string(event), "deployment_info") != -1 {
		container := DeploymentInfoEvent{}
		err = json.Unmarshal(event, &container)
		if err != nil {
			return nil, err
		}

		apps = container.Plan.Target.Apps
	}

	if len(apps) == 0 {
		err = ErrNoApps
	}

	return
}
