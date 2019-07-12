package apps

import (
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Only Marathon apps with this label will be registered in Consul
const MarathonConsulLabel = "consul"

type HealthCheck struct {
	Path                   string `json:"path"`
	PortIndex              int    `json:"portIndex"`
	Port                   int    `json:"port"`
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
	Labels map[string]string `json:"labels"`
}

type Container struct {
	PortMappings []PortDefinition `json:"portMappings"`
}

type appWrapper struct {
	App App `json:"app"`
}

type Apps struct {
	Apps []*App `json:"apps"`
}

type App struct {
	Container       Container         `json:"container"`
	Labels          map[string]string `json:"labels"`
	HealthChecks    []HealthCheck     `json:"healthChecks"`
	ID              AppID             `json:"id"`
	Tasks           []Task            `json:"tasks"`
	PortDefinitions []PortDefinition  `json:"portDefinitions"`
}

// Marathon Application Id (aka PathId)
// Usually in the form of /rootGroup/subGroup/subSubGroup/name
// allowed characters: lowercase letters, digits, hyphens, slash
type AppID string

func (id AppID) String() string {
	return string(id)
}

func (app App) IsConsulApp() bool {
	_, ok := app.Labels[MarathonConsulLabel]
	return ok
}

func (app App) labelsToRawName(labels map[string]string) string {
	if value, ok := labels[MarathonConsulLabel]; ok && !isSpecialConsulNameValue(value) {
		return value
	}
	return app.ID.String()
}

func isSpecialConsulNameValue(name string) bool {
	return name == "true" || name == ""
}

func ParseApps(jsonBlob []byte) ([]*App, error) {
	apps := &Apps{}
	err := json.Unmarshal(jsonBlob, apps)

	return apps.Apps, err
}

func ParseApp(jsonBlob []byte) (*App, error) {
	wrapper := &appWrapper{}
	err := json.Unmarshal(jsonBlob, wrapper)

	return &wrapper.App, err
}

type RegistrationIntent struct {
	Name string
	Port int
	Tags []string
}

func (app App) RegistrationIntentsNumber() int {
	if !app.IsConsulApp() {
		return 0
	}

	definitions := app.findConsulPortDefinitions()
	if len(definitions) == 0 {
		return 1
	}

	return len(definitions)
}

func (app App) RegistrationIntents(task *Task, nameSeparator string) []RegistrationIntent {
	taskPortsCount := len(task.Ports)
	definitions := app.findConsulPortDefinitions()
	commonTags := labelsToTags(app.Labels)
	if len(definitions) == 0 && taskPortsCount != 0 {
		return []RegistrationIntent{
			{
				Name: app.labelsToName(app.Labels, nameSeparator),
				Port: task.Ports[0],
				Tags: commonTags,
			},
		}
	}

	var intents []RegistrationIntent
	for _, d := range definitions {
		if d.Index >= taskPortsCount {
			log.WithField("Id", task.ID.String()).Warnf("Port index (%d) out of bounds should be from range [0,%d)", d.Index, taskPortsCount)
			continue
		}
		intents = append(intents, RegistrationIntent{
			Name: app.labelsToName(d.Labels, nameSeparator),
			Port: task.Ports[d.Index],
			Tags: append(labelsToTags(d.Labels), commonTags...),
		})
	}
	return intents
}

func marathonAppNameToServiceName(name string, nameSeparator string) string {
	return strings.Replace(strings.Trim(strings.TrimSpace(name), "/"), "/", nameSeparator, -1)
}

func labelsToTags(labels map[string]string) []string {
	tags := make([]string, 0, len(labels))
	for key, value := range labels {
		if value == "tag" {
			tags = append(tags, key)
		}
	}
	return tags
}

func (app App) labelsToName(labels map[string]string, nameSeparator string) string {
	appConsulName := app.labelsToRawName(labels)
	serviceName := marathonAppNameToServiceName(appConsulName, nameSeparator)
	if serviceName == "" {
		log.WithField("AppId", app.ID.String()).WithField("ConsulServiceName", appConsulName).
			Warn("Warning! Invalid Consul service name provided for app. Will use default app name instead.")
		return marathonAppNameToServiceName(app.ID.String(), nameSeparator)
	}
	return serviceName
}

type indexedPortDefinition struct {
	Index  int
	Labels map[string]string
}

func (app App) findConsulPortDefinitions() []indexedPortDefinition {
	var definitions []indexedPortDefinition
	for i, d := range app.extractPortDefinitions() {
		if _, ok := d.Labels[MarathonConsulLabel]; ok {
			definitions = append(definitions, indexedPortDefinition{
				Index:  i,
				Labels: d.Labels,
			})
		}
	}

	return definitions
}

// Deprecated: Allows for backward compatibility with Marathons' network API
// PortDefinitions are deprecated in favor of Marathons' new PortMappings
// see https://github.com/mesosphere/marathon/pull/5391
func (app App) extractPortDefinitions() []PortDefinition {
	var appPortDefinitions []PortDefinition
	if len(app.Container.PortMappings) > 0 {
		appPortDefinitions = app.Container.PortMappings
	} else {
		appPortDefinitions = app.PortDefinitions
	}

	return appPortDefinitions
}
