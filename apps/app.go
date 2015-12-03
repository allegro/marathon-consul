package apps

import (
	"github.com/CiscoCloud/marathon-consul/tasks"
)

type HealthCheck struct {
	Path                   string `json:"path"`
	PortIndex              int    `json:"portIndex"`
	Protocol               string `json:"protocol"`
	GracePeriodSeconds     int    `json:"gracePeriodSeconds"`
	IntervalSeconds        int    `json:"intervalSeconds"`
	TimeoutSeconds         int    `json:"timeoutSeconds"`
	MaxConsecutiveFailures int    `json:"maxConsecutiveFailures"`
}

type AppWrapper struct {
	App App `json:"app"`
}

type App struct {
	Labels       map[string]string `json:"labels"`
	HealthChecks []HealthCheck     `json:"healthChecks"`
	ID           string            `json:"id"`
	Tasks        []tasks.Task      `json:"tasks"`
}
