package utils

import (
	"fmt"
	"strings"

	"github.com/allegro/marathon-consul/apps"
)

func ConsulApp(name string, instances int) *apps.App {
	return app(name, instances, true, 0)
}

func ConsulAppWithUnhealthyInstances(name string, instances int, unhealthyInstances int) *apps.App {
	return app(name, instances, true, unhealthyInstances)
}

func NonConsulApp(name string, instances int) *apps.App {
	return app(name, instances, false, 0)
}

func app(name string, instances int, consul bool, unhealthyInstances int) *apps.App {
	var appTasks []apps.Task
	for i := 0; i < instances; i++ {
		task := apps.Task{
			AppID: apps.AppID(name),
			ID:    apps.TaskID(fmt.Sprintf("%s.%d", strings.Replace(strings.Trim(name, "/"), "/", "_", -1), i)),
			Ports: []int{8080 + i},
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
		labels[apps.MarathonConsulLabel] = "true"
	}

	return &apps.App{
		ID:     apps.AppID(name),
		Tasks:  appTasks,
		Labels: labels,
	}
}
