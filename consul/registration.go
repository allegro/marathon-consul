package consul

import (
	"github.com/allegro/marathon-consul/apps"
	log "github.com/Sirupsen/logrus"
	"strings"
)

type RegistrationIntent struct {
	Name string
	Port int
	Tags []string
}

func toRegistrationIntent(task *apps.Task, app *apps.App, nameSeparator string) *RegistrationIntent {
	name := serviceName(app, nameSeparator)
	portIndex := 0
	tags := labelsToTags(app.Labels)

	portDefIndex, portDef, portDefFound := findConsulPortDefinition(app)
	if portDefFound {
		name = labelsToName(app, portDef.Labels, nameSeparator)
		portIndex = portDefIndex
		tags = append(tags, labelsToTags(portDef.Labels)...)
	}

	return &RegistrationIntent{
		Name: name,
		Port: task.Ports[portIndex],
		Tags: tags,
	}
}

func serviceName(app *apps.App, nameSeparator string) string {
	return labelsToName(app, app.Labels, nameSeparator)
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

func labelsToName(app *apps.App, labels map[string]string, nameSeparator string) string {
	appConsulName := app.LabelsToConsulName(labels)
	serviceName := marathonAppNameToServiceName(appConsulName, nameSeparator)
	if serviceName == "" {
		log.WithField("AppId", app.ID.String()).WithField("ConsulServiceName", appConsulName).
		Warn("Warning! Invalid Consul service name provided for app. Will use default app name instead.")
		return marathonAppNameToServiceName(app.ID.String(), nameSeparator)
	}
	return serviceName
}

func findConsulPortDefinition(app *apps.App) (int, apps.PortDefinition, bool) {
	for i, d := range app.PortDefinitions {
		if _, ok := d.Labels[apps.MARATHON_CONSUL_LABEL]; ok {
			return i, d, true
		}
	}
	return -1, apps.PortDefinition{}, false
}
