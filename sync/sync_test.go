package sync

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	. "github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestSyncJob_ShouldSyncOnLeadership(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("app1", 1)
	marathon := marathon.MarathonerStubWithLeaderForApps("current.leader:8080", app)
	services := newConsulServicesMock()
	sync := New(Config{
		Enabled:  true,
		Interval: 10 * time.Millisecond,
		Leader:   "current.leader:8080",
	}, marathon, services)

	// when
	ticker := sync.StartSyncServicesJob()

	// then
	ticker.Stop()
	select {
	case <-time.After(15 * time.Millisecond):
		assert.Equal(t, 1, services.RegistrationsCount(app.Tasks[0].ID.String()))
	}
}

func TestSyncJob_ShouldNotSyncWhenDisabled(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("app1", 1)
	marathon := marathon.MarathonerStubWithLeaderForApps("current.leader:8080", app)
	services := newConsulServicesMock()
	sync := New(Config{
		Enabled:  false,
		Interval: 10 * time.Millisecond,
		Leader:   "current.leader:8080",
	}, marathon, services)

	// when
	ticker := sync.StartSyncServicesJob()

	// then
	assert.Nil(t, ticker)
	assert.Equal(t, 0, services.RegistrationsCount(app.Tasks[0].ID.String()))
}

func TestSyncJob_ShouldDefaultLeaderConfigurationToResolvedHostname(t *testing.T) {
	t.Parallel()
	// given
	hostname, _ := os.Hostname()
	app := ConsulApp("app1", 1)
	marathon := marathon.MarathonerStubWithLeaderForApps(fmt.Sprintf("%s:8080", hostname), app)
	services := newConsulServicesMock()
	sync := New(Config{
		Enabled:  true,
		Interval: 10 * time.Millisecond,
	}, marathon, services)

	// when
	ticker := sync.StartSyncServicesJob()

	// then
	ticker.Stop()
	select {
	case <-time.After(15 * time.Millisecond):
		assert.Equal(t, 1, services.RegistrationsCount(app.Tasks[0].ID.String()))
	}
}

func TestSyncServices_ShouldNotSyncOnNoForceNorLeaderSpecified(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("app1", 1)
	marathon := marathon.MarathonerStubWithLeaderForApps("localhost:8080", app)
	services := newConsulServicesMock()
	sync := New(Config{}, marathon, services)

	// when
	ticker := sync.StartSyncServicesJob()

	// then
	assert.Nil(t, ticker)
	assert.Zero(t, services.RegistrationsCount(app.Tasks[0].ID.String()))
}

func TestSyncServices_ShouldNotSyncOnNoLeadership(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("app1", 1)
	marathon := marathon.MarathonerStubWithLeaderForApps("leader:8080", app)
	services := newConsulServicesMock()
	sync := New(Config{Leader: "different.node:8090"}, marathon, services)

	// when
	err := sync.SyncServices()

	// then
	assert.NoError(t, err)
	assert.Zero(t, services.RegistrationsCount(app.Tasks[0].ID.String()))
}

func TestSyncServices_ShouldSyncOnForceWithoutLeadership(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("app1", 1)
	marathon := marathon.MarathonerStubWithLeaderForApps("leader:8080", app)
	services := newConsulServicesMock()
	sync := New(Config{Leader: "different.node:8090", Force: true}, marathon, services)

	// when
	err := sync.SyncServices()

	// then
	assert.NoError(t, err)
	assert.Equal(t, 1, services.RegistrationsCount(app.Tasks[0].ID.String()))
}

type ConsulServicesMock struct {
	sync.RWMutex
	registrations map[string]int
}

func newConsulServicesMock() *ConsulServicesMock {
	return &ConsulServicesMock{
		registrations: make(map[string]int),
	}
}

func (c *ConsulServicesMock) GetServices(name string) ([]*consulapi.CatalogService, error) {
	return nil, nil
}

func (c *ConsulServicesMock) GetAllServices() ([]*consulapi.CatalogService, error) {
	return nil, nil
}

func (c *ConsulServicesMock) Register(task *apps.Task, app *apps.App) error {
	c.Lock()
	defer c.Unlock()
	c.registrations[task.ID.String()]++
	return nil
}

func (c *ConsulServicesMock) RegistrationsCount(instanceId string) int {
	c.RLock()
	defer c.RUnlock()
	return c.registrations[instanceId]
}

func (c *ConsulServicesMock) ServiceName(app *apps.App) string {
	return ""
}

func (c *ConsulServicesMock) Deregister(serviceId apps.TaskId, agent string) error {
	return nil
}

func (c *ConsulServicesMock) GetAgent(agentAddress string) (*consulapi.Client, error) {
	return nil, nil
}

func TestSyncAppsFromMarathonToConsul(t *testing.T) {
	t.Parallel()
	// given
	marathoner := marathon.MarathonerStubForApps(
		ConsulApp("app1", 2),
		ConsulApp("app2", 1),
		NonConsulApp("app3", 1),
	)

	consul := consul.NewConsulStub()
	marathonSync := newSyncWithDefaultConfig(marathoner, consul)

	// when
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 3, len(services))
	for _, s := range services {
		assert.NotEqual(t, "app3", s.ServiceName)
	}
}

func TestSyncAppsFromMarathonToConsul_CustomServiceName(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("app1", 3)
	app.Labels["consul"] = "customName"
	marathoner := marathon.MarathonerStubForApps(app)

	consul := consul.NewConsulStub()
	marathonSync := newSyncWithDefaultConfig(marathoner, consul)

	// when
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 3, len(services))
	assert.Equal(t, "customName", services[0].ServiceName)
}

func TestRemoveInvalidServicesFromConsul(t *testing.T) {
	t.Parallel()
	// given
	marathoner := marathon.MarathonerStubForApps(
		ConsulApp("app1-invalid", 1),
		ConsulApp("app2", 1),
	)
	consul := consul.NewConsulStub()
	marathonSync := newSyncWithDefaultConfig(marathoner, consul)
	marathonSync.SyncServices()

	// when
	marathoner = marathon.MarathonerStubForApps(
		ConsulApp("app2", 1),
	)
	marathonSync = newSyncWithDefaultConfig(marathoner, consul)
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 1, len(services))
	assert.Equal(t, "app2", services[0].ServiceName)
}

func TestRemoveInvalidServicesFromConsul_WithCustomServiceName(t *testing.T) {
	t.Parallel()
	// given
	invalidApp := ConsulApp("app1-invalid", 1)
	invalidApp.Labels["consul"] = "customName"
	marathoner := marathon.MarathonerStubForApps(
		invalidApp,
		ConsulApp("app2", 1),
	)
	consul := consul.NewConsulStub()
	marathonSync := newSyncWithDefaultConfig(marathoner, consul)

	// when
	marathonSync.SyncServices()

	// then
	customNameServices, _ := consul.GetServices("customName")
	assert.Equal(t, 1, len(customNameServices))

	// when
	marathoner = marathon.MarathonerStubForApps(
		ConsulApp("app2", 1),
	)
	marathonSync = newSyncWithDefaultConfig(marathoner, consul)
	marathonSync.SyncServices()

	// then
	customNameServices, _ = consul.GetServices("customName")
	assert.Equal(t, 0, len(customNameServices))

	services, _ := consul.GetAllServices()
	assert.Equal(t, 1, len(services))
	assert.Equal(t, "app2", services[0].ServiceName)
}

func TestSyncOnlyHealthyServices(t *testing.T) {
	t.Parallel()
	// given
	marathoner := marathon.MarathonerStubForApps(
		ConsulApp("app1", 1),
		ConsulAppWithUnhealthyInstances("app2-one-unhealthy", 2, 1),
		ConsulAppWithUnhealthyInstances("app3-all-unhealthy", 2, 2),
	)
	consul := consul.NewConsulStub()
	marathonSync := newSyncWithDefaultConfig(marathoner, consul)

	// when
	marathonSync.SyncServices()

	// then
	services, _ := consul.GetAllServices()
	assert.Equal(t, 2, len(services))
	for _, s := range services {
		assert.NotEqual(t, "app3-all-unhealthy", s.ServiceName)
	}
}

func TestSync_WithRegisteringProblems(t *testing.T) {
	t.Parallel()
	// given
	marathon := marathon.MarathonerStubForApps(ConsulApp("/test/app", 3))
	consul := consul.NewConsulStub()
	consul.ErrorServices["test_app.1"] = fmt.Errorf("Problem on registration")
	sync := newSyncWithDefaultConfig(marathon, consul)
	// when
	err := sync.SyncServices()
	services, _ := consul.GetAllServices()
	// then
	assert.NoError(t, err)
	assert.Len(t, services, 2)
}

func TestSync_WithDeregisteringProblems(t *testing.T) {
	t.Parallel()
	// given
	marathon := marathon.MarathonerStubForApps()
	consulStub := consul.NewConsulStub()
	notMarathonApp := ConsulApp("/not/marathon", 1)
	for _, task := range notMarathonApp.Tasks {
		consulStub.Register(&task, notMarathonApp)
		consulStub.ErrorServices[task.ID] = fmt.Errorf("Problem on deregistration")
	}
	sync := newSyncWithDefaultConfig(marathon, consulStub)
	// when
	err := sync.SyncServices()
	services, _ := consulStub.GetAllServices()
	// then
	assert.NoError(t, err)
	assert.Len(t, services, 1)
}

func TestSync_WithMarathonProblems(t *testing.T) {
	t.Parallel()
	// given
	marathon := errorMarathon{}
	sync := newSyncWithDefaultConfig(marathon, nil)
	// when
	err := sync.SyncServices()
	// then
	assert.Error(t, err)
}

func TestSync_WithConsulProblems(t *testing.T) {
	t.Parallel()
	// given
	marathon := marathon.MarathonerStubForApps(ConsulApp("/test/app", 3))
	consul := errorConsul{}
	sync := newSyncWithDefaultConfig(marathon, consul)
	// when
	err := sync.SyncServices()
	// then
	assert.Error(t, err)
}

func newSyncWithDefaultConfig(marathon marathon.Marathoner, service consul.ConsulServices) *Sync {
	return New(Config{Enabled: true, Leader: "localhost:8080"}, marathon, service)
}

func TestSync_AddingAgentsFromMarathonTasks(t *testing.T) {
	t.Parallel()

	consulServer := consul.CreateConsulTestServer(t)
	defer consulServer.Stop()

	consulServices := consul.New(consul.ConsulConfig{
		Port: fmt.Sprintf("%d", consulServer.Config.Ports.HTTP),
		Tag:  "marathon",
	})
	app := ConsulApp("serviceA", 2)
	app.Tasks[0].Host = consulServer.Config.Bind
	app.Tasks[1].Host = consulServer.Config.Bind
	marathon := marathon.MarathonerStubWithLeaderForApps("localhost:8080", app)
	sync := New(Config{Leader: "localhost:8080"}, marathon, consulServices)

	// when
	err := sync.SyncServices()

	// then
	assert.NoError(t, err)

	// when
	services, err := consulServices.GetAllServices()

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 2)

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.ServiceName] = struct{}{}
	}
	assert.Len(t, serviceNames, 1)
	assert.Contains(t, serviceNames, "serviceA")
}
