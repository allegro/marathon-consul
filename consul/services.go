package consul

import (
	consulapi "github.com/hashicorp/consul/api"

	"fmt"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/tasks"
	"net/url"
	"strconv"
)

func MarathonTaskToConsulService(task tasks.Task, healthChecks []apps.HealthCheck, labels map[string]string) *consulapi.AgentServiceRegistration {
	return &consulapi.AgentServiceRegistration{
		ID:      task.ID.String(),
		Name:    task.AppID.ConsulServiceName(),
		Port:    task.Ports[0],
		Address: task.Host,
		Tags:    marathonLabelsToConsulTags(labels),
		Check:   marathonToConsulCheck(task, healthChecks),
	}
}

func IsTaskHealthy(healthChecksResults []tasks.HealthCheckResult) bool {
	if len(healthChecksResults) < 1 {
		return false
	}
	register := true
	for _, healthCheckResult := range healthChecksResults {
		register = register && healthCheckResult.Alive
	}
	return register
}

// Takes first HTTP check and convert it to consul healtcheck
// Returns empty check when there is no HTTP check
func marathonToConsulCheck(task tasks.Task, healthChecks []apps.HealthCheck) *consulapi.AgentServiceCheck {
	//	TODO: Handle all types of checks
	for _, check := range healthChecks {
		if check.Protocol == "HTTP" {
			return &consulapi.AgentServiceCheck{
				HTTP: (&url.URL{
					Scheme: "http",
					Host:   task.Host + ":" + strconv.Itoa(task.Ports[check.PortIndex]),
					Path:   check.Path,
				}).String(),
				Interval: fmt.Sprintf("%ds", check.IntervalSeconds),
				Timeout:  fmt.Sprintf("%ds", check.TimeoutSeconds),
			}
		}
	}
	return nil
}

// Extract labels keys with value tag and return as slice
func marathonLabelsToConsulTags(labels map[string]string) []string {
	tags := []string{"marathon"}
	for key, value := range labels {
		if value == "tag" {
			tags = append(tags, key)
		}
	}
	return tags
}
