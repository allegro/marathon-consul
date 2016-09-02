package apps

import (
	"encoding/json"
	"strings"

	log "github.com/Sirupsen/logrus"
	"reflect"
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

func (app *App) HasSameConsulNamesAs(other *App) bool {
	return reflect.DeepEqual(rawConsulNames(app), rawConsulNames(other))
}

func rawConsulNames(app *App) []string {
	var names []string
	for _, i := range app.RegistrationIntents("/") {
		names = append(names, i.Name)
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
	Name      string
	PortIndex int
	Tags      []string
}

func (app *App) RegistrationIntents(nameSeparator string) []*RegistrationIntent {
	commonTags := labelsToTags(app.Labels)

	definitions := app.findConsulPortDefinitions()
	if len(definitions) == 0 {
		return []*RegistrationIntent{
			&RegistrationIntent{
				Name: app.labelsToName(app.Labels, nameSeparator),
				PortIndex: 0,
				Tags: commonTags,
			},
		}
	}

	var intents []*RegistrationIntent
	for _, d := range definitions {
		intents = append(intents, &RegistrationIntent{
			Name: app.labelsToName(d.Labels, nameSeparator),
			PortIndex: d.PortIndex,
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
	PortIndex int
	Labels    map[string]string
}

func (app *App) findConsulPortDefinitions() []IndexedPortDefinition {
	var definitions []IndexedPortDefinition
	for i, d := range app.PortDefinitions {
		if _, ok := d.Labels[MARATHON_CONSUL_LABEL]; ok {
			definitions = append(definitions, IndexedPortDefinition{
				PortIndex: i,
				Labels:    d.Labels,
			})
		}
	}
	return definitions
}
