package utils

import (
	"fmt"
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/tasks"
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
	var appTasks []tasks.Task
	for i := 0; i < instances; i++ {
		task := tasks.Task{
			AppID: name,
			ID:    fmt.Sprintf("%s.%d", name, i),
			Ports: []int{8080 + i},
			Host:  "",
		}
		if unhealthyInstances > 0 {
			unhealthyInstances--
		} else {
			task.HealthCheckResults = []tasks.HealthCheckResult{
				tasks.HealthCheckResult{
					Alive: true,
				},
			}
		}
		appTasks = append(appTasks, task)
	}

	labels := make(map[string]string)
	if consul {
		labels["consul"] = "true"
	}

	return &apps.App{
		ID:     name,
		Tasks:  appTasks,
		Labels: labels,
	}
}
