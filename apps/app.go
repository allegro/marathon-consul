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
	Port   int               `json:"port"`
	Labels map[string]string `json:"labels"`
}

type AppWrapper struct {
	App App `json:"app"`
}

type AppsResponse struct {
	Apps []*App `json:"apps"`
}

type App struct {
	Labels          map[string]string `json:"labels"`
	HealthChecks    []HealthCheck     `json:"healthChecks"`
	ID              AppId             `json:"id"`
	Tasks           []Task            `json:"tasks"`
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

func (app *App) HasSameConsulNamesAs(other *App) bool {
	thisNames := app.ConsulNames(".")
	otherNames := other.ConsulNames(".")

	if len(thisNames) != len(otherNames) {
		return false
	}

	for i, name := range thisNames {
		if name != otherNames[i] {
			return false
		}
	}
	return true
}

func (app *App) ConsulNames(separator string) []string {
	definitions := app.findConsulPortDefinitions()

	if len(definitions) == 0 {
		return []string{app.labelsToName(app.Labels, separator)}
	}

	var names []string
	for _, d := range definitions {
		names = append(names, app.labelsToName(d.Labels, separator))
	}
	return names
}

func (app *App) labelsToRawName(labels map[string]string) string {
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

func (app *App) RegistrationIntents(task *Task, nameSeparator string) []*RegistrationIntent {
	commonTags := labelsToTags(app.Labels)

	definitions := app.findConsulPortDefinitions()
	if len(definitions) == 0 {
		return []*RegistrationIntent{
			&RegistrationIntent{
				Name: app.labelsToName(app.Labels, nameSeparator),
				Port: task.Ports[0],
				Tags: commonTags,
			},
		}
	}

	var intents []*RegistrationIntent
	for _, d := range definitions {
		intents = append(intents, &RegistrationIntent{
			Name: app.labelsToName(d.Labels, nameSeparator),
			Port: d.toPort(task),
			Tags: append(commonTags, labelsToTags(d.Labels)...),
		})
	}
	return intents
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
	appConsulName := app.labelsToRawName(labels)
	serviceName := marathonAppNameToServiceName(appConsulName, nameSeparator)
	if serviceName == "" {
		log.WithField("AppId", app.ID.String()).WithField("ConsulServiceName", appConsulName).
		Warn("Warning! Invalid Consul service name provided for app. Will use default app name instead.")
		return marathonAppNameToServiceName(app.ID.String(), nameSeparator)
	}
	return serviceName
}

type IndexedPortDefinition struct {
	Index  int
	Port   int
	Labels map[string]string
}

func (i *IndexedPortDefinition) toPort(task *Task) int {
	if i.Port == 0 {
		return task.Ports[i.Index]
	}
	return i.Port
}

func (app *App) findConsulPortDefinitions() []IndexedPortDefinition {
	var definitions []IndexedPortDefinition
	for i, d := range app.PortDefinitions {
		if _, ok := d.Labels[MARATHON_CONSUL_LABEL]; ok {
			definitions = append(definitions, IndexedPortDefinition{
				Index:  i,
				Port:   d.Port,
				Labels: d.Labels,
			})
		}
	}
	return definitions
}
