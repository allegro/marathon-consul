package marathon

import (
	"fmt"
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSyncAppsFromMarathonToConsul(t *testing.T) {
	// given
	marathoner := MarathonerStubForApps(
		consulApp("app1", 2),
		consulApp("app2", 1),
		nonConsulApp("app3", 1),
	)

	consul := consul.NewConsulStub()
	marathonSync := NewMarathonSync(marathoner, consul)

	// when
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 3, len(services))
	for _, s := range services {
		assert.NotEqual(t, "app3", s.ServiceName)
	}
}

func TestRemoveInvalidServicesFromConsul(t *testing.T) {
	// given
	marathoner := MarathonerStubForApps(
		consulApp("app1-invalid", 1),
		consulApp("app2", 1),
	)
	consul := consul.NewConsulStub()
	marathonSync := NewMarathonSync(marathoner, consul)
	marathonSync.SyncServices()

	// when
	marathoner = MarathonerStubForApps(
		app("app2", 1, true),
	)
	marathonSync = NewMarathonSync(marathoner, consul)
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 1, len(services))
	assert.Equal(t, "app2", services[0].ServiceName)
}

func consulApp(name string, instances int) *apps.App {
	return app(name, instances, true)
}

func nonConsulApp(name string, instances int) *apps.App {
	return app(name, instances, false)
}

func app(name string, instances int, consul bool) *apps.App {
	var appTasks []tasks.Task
	for i := 0; i < instances; i++ {
		appTasks = append(appTasks, tasks.Task{
			AppID: name,
			ID:    fmt.Sprintf("%s.%d", name, i),
			Ports: []int{8080 + i},
			Host:  "",
		})
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
