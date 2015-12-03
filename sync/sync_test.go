package sync

import (
	"fmt"
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/marathon"
	"github.com/CiscoCloud/marathon-consul/tasks"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"testing"

	"time"
)

func TestSyncJob(t *testing.T) {
	t.Parallel()
	// given
	app := consulApp("app1", 1)
	marathon := marathon.MarathonerStubForApps(app)
	services := newConsulServicesMock()
	sync := New(marathon, services)

	// when
	ticker := sync.StartSyncServicesJob(10 * time.Millisecond)

	// then
	select {
	case <-time.After(15 * time.Millisecond):
		ticker.Stop()
		assert.Equal(t, 2, services.RegistrationsCount(app.Tasks[0].ID))
	}
}

type ConsulServicesMock struct {
	registrations map[string]int
}

func newConsulServicesMock() *ConsulServicesMock {
	return &ConsulServicesMock{
		registrations: make(map[string]int),
	}
}

func (c *ConsulServicesMock) GetAllServices() ([]*consulapi.CatalogService, error) {
	return nil, nil
}

func (c *ConsulServicesMock) Register(service *consulapi.AgentServiceRegistration) error {
	c.registrations[service.ID]++
	return nil
}

func (c *ConsulServicesMock) RegistrationsCount(instanceId string) int {
	return c.registrations[instanceId]
}

func (c *ConsulServicesMock) Deregister(serviceId string, agent string) error {
	return nil
}

func TestSyncAppsFromMarathonToConsul(t *testing.T) {
	// given
	marathoner := marathon.MarathonerStubForApps(
		consulApp("app1", 2),
		consulApp("app2", 1),
		nonConsulApp("app3", 1),
	)

	consul := consul.NewConsulStub()
	marathonSync := New(marathoner, consul)

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
	marathoner := marathon.MarathonerStubForApps(
		consulApp("app1-invalid", 1),
		consulApp("app2", 1),
	)
	consul := consul.NewConsulStub()
	marathonSync := New(marathoner, consul)
	marathonSync.SyncServices()

	// when
	marathoner = marathon.MarathonerStubForApps(
		app("app2", 1, true),
	)
	marathonSync = New(marathoner, consul)
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
