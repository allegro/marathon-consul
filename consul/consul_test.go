package consul

import (
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAgent_WithEmptyHost(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})
	// when
	agent, err := agents.GetAgent("")
	// then
	assert.Error(t, err)
	assert.Nil(t, agent)
}

func TestGetAgent_FullConfig(t *testing.T) {
	t.Parallel()

	// given
	agents := NewAgents(&ConsulConfig{Token: "token", SslEnabled: true,
		Auth: Auth{Enabled: true, Username: "", Password: ""}})

	// when
	agent, err := agents.GetAgent("host")

	// then
	assert.NoError(t, err)
	assert.NotNil(t, agent)
}

func TestGetAllServices_ForEmptyAgents(t *testing.T) {
	t.Parallel()

	// given
	consul := New(ConsulConfig{})

	// when
	services, err := consul.GetAllServices()

	// then
	assert.Error(t, err)
	assert.Nil(t, services)
}

func TestRegister_ForInvalidHost(t *testing.T) {
	t.Parallel()

	// given
	consul := New(ConsulConfig{})
	app := utils.ConsulApp("serviceA", 1)

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.Error(t, err)

	// when
	err = consul.Deregister("service", "")

	// then
	assert.Error(t, err)
}

func TestGetServices(t *testing.T) {
	t.Parallel()
	// create cluster of 2 consul servers
	server1 := CreateConsulTestServer("dc1", t)
	defer server1.Stop()

	server2 := CreateConsulTestServer("dc2", t)
	defer server2.Stop()

	server1.JoinWAN(server2.LANAddr)

	// create client
	consul := ConsulClientAtServer(server1)
	consul.config.Tag = "marathon"

	// given
	// register services in both servers
	server1.AddService("serviceA", "passing", []string{"public", "marathon"})
	server1.AddService("serviceB", "passing", []string{"marathon"})
	server1.AddService("serviceC", "passing", []string{"marathon"})

	server2.AddService("serviceA", "passing", []string{"private", "marathon"})
	server2.AddService("serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetServices("serviceA")

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

func TestGetServices_SelectOnlyTaggedServices(t *testing.T) {
	t.Parallel()
	// create cluster of 2 consul servers
	server1 := CreateConsulTestServer("dc1", t)
	defer server1.Stop()

	server2 := CreateConsulTestServer("dc2", t)
	defer server2.Stop()

	server1.JoinWAN(server2.LANAddr)

	// create client
	consul := ConsulClientAtServer(server1)
	consul.config.Tag = "marathon-mycluster"

	// given
	// register services in both servers
	server1.AddService("serviceA", "passing", []string{"public", "marathon-mycluster"})
	server1.AddService("serviceB", "passing", []string{"marathon"})
	server1.AddService("serviceC", "passing", []string{"marathon"})

	server2.AddService("serviceA", "passing", []string{"private", "marathon"})
	server2.AddService("serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetServices("serviceA")

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Contains(t, services[0].ServiceTags, "marathon-mycluster")

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.ServiceName] = struct{}{}
	}
	assert.Len(t, serviceNames, 1)
	assert.Contains(t, serviceNames, "serviceA")
}

func TestGetAllServices(t *testing.T) {
	t.Parallel()
	// create cluster of 2 consul servers
	server1 := CreateConsulTestServer("dc1", t)
	defer server1.Stop()

	server2 := CreateConsulTestServer("dc2", t)
	defer server2.Stop()

	server1.JoinWAN(server2.LANAddr)

	// create client
	consul := ConsulClientAtServer(server1)
	consul.config.Tag = "marathon"

	// given
	// register services in both servers
	server1.AddService("serviceA", "passing", []string{"public", "marathon"})
	server1.AddService("serviceB", "passing", []string{"marathon"})
	server1.AddService("serviceC", "passing", []string{"zookeeper"})

	server2.AddService("serviceA", "passing", []string{"private", "marathon"})
	server2.AddService("serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetAllServices()

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 3)

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.ServiceName] = struct{}{}
	}
	assert.Len(t, serviceNames, 2)
	assert.Contains(t, serviceNames, "serviceA")
	assert.Contains(t, serviceNames, "serviceB")
}

func TestRegisterServices(t *testing.T) {
	t.Parallel()
	server := CreateConsulTestServer("dc1", t)
	defer server.Stop()

	consul := ConsulClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	app.Tasks[0].Host = server.Config.Bind
	app.Labels["test"] = "tag"

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.NoError(t, err)

	// when
	services, _ := consul.GetAllServices()

	// then
	assert.Len(t, services, 1)
	assert.Equal(t, "serviceA", services[0].ServiceName)
	assert.Equal(t, []string{"marathon", "test"}, services[0].ServiceTags)
}

func TestRegisterServices_CustomServiceName(t *testing.T) {
	t.Parallel()
	server := CreateConsulTestServer("dc1", t)
	defer server.Stop()

	consul := ConsulClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	app.Tasks[0].Host = server.Config.Bind
	app.Labels["test"] = "tag"
	app.Labels["consul"] = "myCustomServiceName"

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.NoError(t, err)

	// when
	services, _ := consul.GetAllServices()

	// then
	assert.Len(t, services, 1)
	assert.Equal(t, "myCustomServiceName", services[0].ServiceName)
	assert.Equal(t, []string{"marathon", "test"}, services[0].ServiceTags)
}

func TestRegisterServices_InvalidCustomServiceName(t *testing.T) {
	t.Parallel()
	server := CreateConsulTestServer("dc1", t)
	defer server.Stop()

	consul := ConsulClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	app.Tasks[0].Host = server.Config.Bind
	app.Labels["test"] = "tag"
	app.Labels["consul"] = " /"

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.NoError(t, err)

	// when
	services, _ := consul.GetAllServices()

	// then
	assert.Len(t, services, 1)
	assert.Equal(t, "serviceA", services[0].ServiceName)
	assert.Equal(t, []string{"marathon", "test"}, services[0].ServiceTags)
}

func TestRegisterServices_shouldReturnErrorOnFailure(t *testing.T) {
	t.Parallel()

	// given
	consul := New(ConsulConfig{Port: "1234"})
	app := utils.ConsulApp("serviceA", 1)

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.Error(t, err)
}

func TestDeregisterServices(t *testing.T) {
	t.Parallel()
	server := CreateConsulTestServer("dc1", t)
	defer server.Stop()

	consul := ConsulClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	server.AddService("serviceA", "passing", []string{"marathon"})
	server.AddService("serviceB", "passing", []string{"marathon"})
	services, _ := consul.GetAllServices()
	assert.Len(t, services, 2)

	// when
	consul.Deregister("serviceA", server.Config.Bind)

	// then
	services, _ = consul.GetAllServices()
	assert.Len(t, services, 1)
	assert.Equal(t, "serviceB", services[0].ServiceName)
}

func TestDeregisterServices_shouldReturnErrorOnFailure(t *testing.T) {
	t.Parallel()
	server := CreateConsulTestServer("dc1", t)
	defer server.Stop()

	consul := ConsulClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	server.AddService("serviceA", "passing", []string{"marathon"})

	// when
	server.Stop()
	err := consul.Deregister("serviceA", server.Config.Bind)

	// then
	assert.Error(t, err)
}

func TestMarathonTaskToConsulServiceMapping_WithNoHttpChecks(t *testing.T) {
	t.Parallel()

	// given
	consul := New(ConsulConfig{})

	app := &apps.App{
		ID: "someApp",
		HealthChecks: []apps.HealthCheck{
			apps.HealthCheck{
				Path:                   "/",
				Protocol:               "TCP",
				PortIndex:              0,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
		},
		Labels: map[string]string{
			"consul": "true",
			"public": "tag",
		},
	}
	task := &apps.Task{
		ID:    "someTask",
		AppID: app.ID,
		Host:  "127.0.0.6",
		Ports: []int{8090, 8443},
	}

	// when
	service := consul.marathonTaskToConsulService(task, app)

	// then
	assert.Equal(t, "127.0.0.6", service.Address)
	assert.Equal(t, 8090, service.Port)
	assert.Nil(t, service.Check)
	assert.Empty(t, service.Checks)
}

func TestMarathonTaskToConsulServiceMapping(t *testing.T) {
	t.Parallel()

	// given
	consul := New(ConsulConfig{Tag: "marathon"})
	app := &apps.App{
		ID: "someApp",
		HealthChecks: []apps.HealthCheck{
			apps.HealthCheck{
				Path:                   "/api/health",
				Protocol:               "HTTP",
				PortIndex:              0,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
		},
		Labels: map[string]string{
			"consul": "true",
			"public": "tag",
		},
	}
	task := &apps.Task{
		ID:    "someTask",
		AppID: app.ID,
		Host:  "127.0.0.6",
		Ports: []int{8090, 8443},
	}

	// when
	service := consul.marathonTaskToConsulService(task, app)

	// then
	assert.Equal(t, "127.0.0.6", service.Address)
	assert.Equal(t, []string{"marathon", "public"}, service.Tags)
	assert.Equal(t, 8090, service.Port)
	assert.NotNil(t, "http://127.0.0.6:8090/api/health", service.Check)
	assert.Empty(t, service.Checks)
	assert.Equal(t, "http://127.0.0.6:8090/api/health", service.Check.HTTP)
	assert.Equal(t, "60s", service.Check.Interval)
}
