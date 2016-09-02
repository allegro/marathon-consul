package apps

import (
	"encoding/json"
	"strings"

	log "github.com/Sirupsen/logrus"
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
		Value string `json:"value"`
	}
}

type PortDefinition struct {
	Labels map[string]string	`json:"labels"`
}

type AppWrapper struct {
	App App `json:"app"`
}

type AppsResponse struct {
	Apps []*App `json:"apps"`
}

type App struct {
	Labels       	map[string]string `json:"labels"`
	HealthChecks 	[]HealthCheck     `json:"healthChecks"`
	ID           	AppId             `json:"id"`
	Tasks        	[]Task            `json:"tasks"`
	PortDefinitions []PortDefinition  `json:"portDefinitions"`
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
	_, portDef, found := app.findConsulPortDefinition()
	if found {
		return app.LabelsToConsulName(portDef.Labels)
	}
	return app.LabelsToConsulName(app.Labels)
}

func (app *App) LabelsToConsulName(labels map[string]string) string {
	if value, ok := labels[MARATHON_CONSUL_LABEL]; ok && !isSpecialConsulNameValue(value) {
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

type RegistrationIntent struct {
	Name string
	Port int
	Tags []string
}

func (app *App) RegistrationIntent(task *Task, nameSeparator string) *RegistrationIntent {
	name := app.serviceName(nameSeparator)
	portIndex := 0
	tags := labelsToTags(app.Labels)

	portDefIndex, portDef, portDefFound := app.findConsulPortDefinition()
	if portDefFound {
		name = app.labelsToName(portDef.Labels, nameSeparator)
		portIndex = portDefIndex
		tags = append(tags, labelsToTags(portDef.Labels)...)
	}

	return &RegistrationIntent{
		Name: name,
		Port: task.Ports[portIndex],
		Tags: tags,
	}
}

func (app *App) serviceName(nameSeparator string) string {
	return app.labelsToName(app.Labels, nameSeparator)
}

func marathonAppNameToServiceName(name string, nameSeparator string) string {
	return strings.Replace(strings.Trim(strings.TrimSpace(name), "/"), "/", nameSeparator, -1)
}

func labelsToTags(labels map[string]string) []string {
	tags := []string{}
	for key, value := range labels {
		if value == "tag" {
			tags = append(tags, key)
		}
	}
	return tags
}

func (app *App) labelsToName(labels map[string]string, nameSeparator string) string {
	appConsulName := app.LabelsToConsulName(labels)
	serviceName := marathonAppNameToServiceName(appConsulName, nameSeparator)
	if serviceName == "" {
		log.WithField("AppId", app.ID.String()).WithField("ConsulServiceName", appConsulName).
		Warn("Warning! Invalid Consul service name provided for app. Will use default app name instead.")
		return marathonAppNameToServiceName(app.ID.String(), nameSeparator)
	}
	return serviceName
}

func (app *App) findConsulPortDefinition() (int, PortDefinition, bool) {
	for i, d := range app.PortDefinitions {
		if _, ok := d.Labels[MARATHON_CONSUL_LABEL]; ok {
			return i, d, true
		}
	}
	return -1, PortDefinition{}, false
}
