package consul

import (
	"github.com/CiscoCloud/mesos-consul/registry"

	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"net/url"
	"strconv"
	"strings"
)

func MarathonTaskToConsulService(task tasks.Task, healthChecks []apps.HealthCheck, labels map[string]string) *registry.Service {
	return &registry.Service{
		ID:      task.ID,
		Name:    appIdToServiceName(task.AppID),
		Port:    task.Ports[0], /*By default app should use its 1st port*/
		Address: task.Host,
		Tags:    marathonLabelsToConsulTags(labels),
		Check:   marathonToConsulCheck(task, healthChecks),
		Agent:   task.Host,
	}
}

func IsTaskHealthy(healthChecksResults []tasks.HealthCheckResult) bool {
	register := true
	for _, healthCheckResult := range healthChecksResults {
		register = register && healthCheckResult.Alive
	}
	return register
}

// Takes first HTTP check and convert it to consul healtcheck
// Returns empty check when there is no HTTP check
func marathonToConsulCheck(task tasks.Task, healthChecks []apps.HealthCheck) *registry.Check {
	//	TODO: Handle all types of checks
	for _, check := range healthChecks {
		if check.Protocol == "HTTP" {
			return &registry.Check{
				HTTP: (&url.URL{
					Scheme: "http",
					Host:   task.Host + ":" + strconv.Itoa(task.Ports[check.PortIndex]),
					Path:   check.Path,
				}).String(),
				Interval: strconv.Itoa(check.IntervalSeconds) + "s",
			}
		}
	}
	return &registry.Check{}
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

func appIdToServiceName(appId string) (serviceId string) {
	serviceId = strings.Replace(strings.Trim(appId, "/"), "/", ".", -1)
	return serviceId
}
