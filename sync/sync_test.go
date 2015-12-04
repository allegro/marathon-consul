package sync

import (
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	. "github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"testing"

	"time"
)

func TestSyncJob(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("app1", 1)
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
		ConsulApp("app1", 2),
		ConsulApp("app2", 1),
		NonConsulApp("app3", 1),
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
		ConsulApp("app1-invalid", 1),
		ConsulApp("app2", 1),
	)
	consul := consul.NewConsulStub()
	marathonSync := New(marathoner, consul)
	marathonSync.SyncServices()

	// when
	marathoner = marathon.MarathonerStubForApps(
		ConsulApp("app2", 1),
	)
	marathonSync = New(marathoner, consul)
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 1, len(services))
	assert.Equal(t, "app2", services[0].ServiceName)
}

func TestSyncOnlyHealthyServices(t *testing.T) {
	// given
	marathoner := marathon.MarathonerStubForApps(
		ConsulApp("app1", 1),
		ConsulAppWithUnhealthyInstances("app2-one-unhealthy", 2, 1),
		ConsulAppWithUnhealthyInstances("app3-all-unhealthy", 2, 2),
	)
	consul := consul.NewConsulStub()
	marathonSync := New(marathoner, consul)

	// when
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 2, len(services))
	for _, s := range services {
		assert.NotEqual(t, "app3-all-unhealthy", s.ServiceName)
	}
}
