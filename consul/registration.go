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
	return &RegistrationIntent{
		Name: serviceName(app, nameSeparator),
		Port: task.Ports[0],
		Tags: labelsToTags(app.Labels),
	}
}

func serviceName(app *apps.App, nameSeparator string) string {
	appConsulName := app.ConsulName()
	serviceName := marathonAppNameToServiceName(appConsulName, nameSeparator)
	if serviceName == "" {
		log.WithField("AppId", app.ID.String()).WithField("ConsulServiceName", appConsulName).
		Warn("Warning! Invalid Consul service name provided for app. Will use default app name instead.")
		return marathonAppNameToServiceName(app.ID.String(), nameSeparator)
	}
	return serviceName
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
