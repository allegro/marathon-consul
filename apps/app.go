package apps

import (
	"encoding/json"
)

// Only Marathon apps with this label will be registered in Consul
const MARATHON_CONSUL_LABEL = "consul"

type HealthCheck struct {
	Path                   string `json:"path"`
	PortIndex              int    `json:"portIndex"`
	Protocol               string `json:"protocol"`
	GracePeriodSeconds     int    `json:"gracePeriodSeconds"`
	IntervalSeconds        int    `json:"intervalSeconds"`
	TimeoutSeconds         int    `json:"timeoutSeconds"`
	MaxConsecutiveFailures int    `json:"maxConsecutiveFailures"`
	Command                struct {
		Value string `json:"value`
	}
}

type AppWrapper struct {
	App App `json:"app"`
}

type AppsResponse struct {
	Apps []*App `json:"apps"`
}

type App struct {
	Labels       map[string]string `json:"labels"`
	HealthChecks []HealthCheck     `json:"healthChecks"`
	ID           AppId             `json:"id"`
	Tasks        []Task            `json:"tasks"`
}

// Marathon Application Id (aka PathId)
// Usually in the form of /rootGroup/subGroup/subSubGroup/name
// allowed characters: lowercase letters, digits, hyphens, slash
type AppId string

func (id AppId) String() string {
	return string(id)
}

func (app *App) IsConsulApp() bool {
	_, ok := app.Labels[MARATHON_CONSUL_LABEL]
	return ok
}

func (app *App) ConsulName() string {
	if value, ok := app.Labels[MARATHON_CONSUL_LABEL]; ok && !isSpecialConsulNameValue(value) {
		return value
	}
	return app.ID.String()
}

func isSpecialConsulNameValue(name string) bool {
	return name == "true" || name == ""
}

func ParseApps(jsonBlob []byte) ([]*App, error) {
	apps := &AppsResponse{}
	err := json.Unmarshal(jsonBlob, apps)

	return apps.Apps, err
}

func ParseApp(jsonBlob []byte) (*App, error) {
	wrapper := &AppWrapper{}
	err := json.Unmarshal(jsonBlob, wrapper)

	return &wrapper.App, err
}
