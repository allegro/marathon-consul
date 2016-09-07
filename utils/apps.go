package utils

import (
	"fmt"
	"strings"

	"github.com/allegro/marathon-consul/apps"
)

func ConsulApp(name string, instances int) *apps.App {
	return app(name, instances, 1, true, 0)
}

func ConsulAppWithUnhealthyInstances(name string, instances int, unhealthyInstances int) *apps.App {
	return app(name, instances, 1, true, unhealthyInstances)
}

func ConsulAppMultipleRegistrations(name string, instances int, registrations int) *apps.App {
	return app(name, instances, registrations, true, 0)
}

func NonConsulApp(name string, instances int) *apps.App {
	return app(name, instances, 1, false, 0)
}

func app(name string, instances int, registrationsPerInstance int, consul bool, unhealthyInstances int) *apps.App {
	var appTasks []apps.Task
	for i := 0; i < instances; i++ {
		var ports []int
		for j := 1; j <= registrationsPerInstance; j++ {
			ports = append(ports, 8080 + (i * j) + j - 1)
		}
		task := apps.Task{
			AppID: apps.AppId(name),
			ID:    apps.TaskId(fmt.Sprintf("%s.%d", strings.Replace(strings.Trim(name, "/"), "/", "_", -1), i)),
			Ports: ports,
			Host:  "localhost",
		}
		if unhealthyInstances > 0 {
			unhealthyInstances--
		} else {
			task.HealthCheckResults = []apps.HealthCheckResult{
				{
					Alive: true,
				},
			}
		}
		appTasks = append(appTasks, task)
	}

	labels := make(map[string]string)
	if consul {
		labels[apps.MARATHON_CONSUL_LABEL] = "true"
	}

	app := &apps.App{
		ID:     apps.AppId(name),
		Tasks:  appTasks,
		Labels: labels,
	}

	if registrationsPerInstance > 1 {
		for i := 0; i < registrationsPerInstance; i++ {
			app.PortDefinitions = append(app.PortDefinitions, apps.PortDefinition{
				Port: 0,
				Labels: map[string]string{"consul": ""},
			})
		}
	}

	return app
}
