package sync

import (
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	. "github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"testing"

	"fmt"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/tasks"
	"os"
	"time"
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
	select {
	case <-time.After(15 * time.Millisecond):
		ticker.Stop()
		assert.Equal(t, 2, services.RegistrationsCount(app.Tasks[0].ID.String()))
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
	select {
	case <-time.After(15 * time.Millisecond):
		assert.Equal(t, 0, services.RegistrationsCount(app.Tasks[0].ID.String()))
	}
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
	select {
	case <-time.After(15 * time.Millisecond):
		ticker.Stop()
		assert.Equal(t, 2, services.RegistrationsCount(app.Tasks[0].ID.String()))
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

func (c *ConsulServicesMock) Deregister(serviceId tasks.Id, agent string) error {
	return nil
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

func TestRemoveInvalidServicesFromConsul(t *testing.T) {
	t.Parallel()
	// given
	marathoner := marathon.MarathonerStubForApps(
		ConsulApp("app1-invalid", 1),
		ConsulApp("app2", 1),
	)
	consul := consul.NewConsulStub()
	marathonSync := New(Config{}, marathoner, consul)
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
	consul.ErrorServices["/test/app.1"] = fmt.Errorf("Problem on registration")
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
		consulStub.Register(consul.MarathonTaskToConsulService(task, notMarathonApp.HealthChecks, notMarathonApp.Labels))
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
	marathon := marathon.MarathonerStubForApps()
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

type errorMarathon struct {
}

func (m errorMarathon) Apps() ([]*apps.App, error) {
	return nil, fmt.Errorf("Error")
}

func (m errorMarathon) App(id tasks.AppId) (*apps.App, error) {
	return nil, fmt.Errorf("Error")
}

func (m errorMarathon) Tasks(appId tasks.AppId) ([]*tasks.Task, error) {
	return nil, fmt.Errorf("Error")
}

func (m errorMarathon) Leader() (string, error) {
	return "", fmt.Errorf("Error")
}

type errorConsul struct {
}

func (c errorConsul) GetAllServices() ([]*consulapi.CatalogService, error) {
	return nil, fmt.Errorf("Error occured")
}
func (c errorConsul) Register(service *consulapi.AgentServiceRegistration) error {
	return fmt.Errorf("Error occured")

}
func (c errorConsul) Deregister(serviceId tasks.Id, agent string) error {
	return fmt.Errorf("Error occured")
}
