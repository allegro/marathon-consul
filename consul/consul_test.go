package consul

import (
	"fmt"
	"testing"
	"time"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/service"
	timeutil "github.com/allegro/marathon-consul/time"
	"github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestGetAgent_WithEmptyHost(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})
	// when
	agent, err := agents.GetAgent("")
	// then
	assert.Error(t, err)
	assert.Nil(t, agent)
}

func TestGetAgent_FullConfig(t *testing.T) {
	t.Parallel()

	// given
	agents := NewAgents(&Config{Token: "token", SslEnabled: true,
		Auth: Auth{Enabled: true, Username: "", Password: ""}, Timeout: timeutil.Interval{Duration: time.Second}})

	// when
	agent, err := agents.GetAgent("127.23.23.23")

	// then
	assert.NoError(t, err)
	assert.NotNil(t, agent)
}

func TestGetAllServices_ForEmptyAgents(t *testing.T) {
	t.Parallel()

	// given
	consul := New(Config{})

	// when
	services, err := consul.GetAllServices()

	// then
	assert.Error(t, err)
	assert.Nil(t, services)
}

func TestRegister_ForInvalidHost(t *testing.T) {
	t.Parallel()

	// given
	consul := New(Config{})
	app := utils.ConsulApp("serviceA", 1)

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.Error(t, err)

	// when
	err = consul.Deregister(&service.Service{
		ID:   service.ServiceId("someId"),
		Name: "service",
		Tags: []string{},
		RegisteringAgentAddress: "",
	})

	// then
	assert.Error(t, err)
}

func TestGetServices(t *testing.T) {
	t.Parallel()
	// create cluster of 2 consul servers
	server1 := CreateTestServer(t)
	defer server1.Stop()

	server2 := CreateTestServer(t)
	defer server2.Stop()

	server1.JoinWAN(t, server2.LANAddr)

	// create client
	consul := ClientAtServer(server1)
	consul.config.Tag = "marathon"

	// given
	// register services in both servers
	server1.AddService(t, "serviceA", "passing", []string{"public", "marathon"})
	server1.AddService(t, "serviceB", "passing", []string{"marathon"})
	server1.AddService(t, "serviceC", "passing", []string{"marathon"})

	server2.AddService(t, "serviceA", "passing", []string{"private", "marathon"})
	server2.AddService(t, "serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetServices("serviceA")

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 2)

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.Name] = struct{}{}
	}
	assert.Len(t, serviceNames, 1)
	assert.Contains(t, serviceNames, "serviceA")
}

func TestGetService_FailingAgent_GivingUp(t *testing.T) {
	t.Parallel()
	server1 := CreateTestServer(t)
	defer server1.Stop()

	// create client
	consul := FailingClient()

	// when
	services, err := consul.GetServices("serviceA")

	// then
	assert.EqualError(t, err, "An error occurred getting services from Consul. Giving up")
	assert.Nil(t, services)
}

func TestServiceHasEnabledTagOverriding(t *testing.T) {
	t.Parallel()

	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)

	app := utils.ConsulApp("serviceA", 1)
	app.Tasks[0].Host = server.Config.Bind
	app.Labels["test"] = "tag"

	err := consul.Register(&app.Tasks[0], app)

	assert.NoError(t, err)

	consul_services, err := consul.GetServices("serviceA")
	services := make(map[string]*service.Service)
	for _, s := range consul_services {
		services[s.Name] = s
	}

	assert.NoError(t, err)
	assert.True(t, services["serviceA"].EnableTagOverride)
}

func TestGetServices_RemovingFailingAgentsAndRetrying(t *testing.T) {
	t.Parallel()
	// create server
	server1 := CreateTestServer(t)
	defer server1.Stop()

	// create client
	consul := ClientAtServer(server1)
	consul.config.Tag = "marathon"
	consul.config.RequestRetries = 10

	// given
	server1.AddService(t, "serviceA", "passing", []string{"public", "marathon"})
	server1.AddService(t, "serviceB", "passing", []string{"marathon"})

	// add failing clients
	for i := uint32(2); i < consul.config.RequestRetries; i++ {
		consul.AddAgent(fmt.Sprintf("127.0.0.%d", i))
	}

	// when
	services, err := consul.GetServices("serviceA")

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 1)
}

func TestGetServices_SelectOnlyTaggedServices(t *testing.T) {
	t.Parallel()
	// create cluster of 2 consul servers
	server1 := CreateTestServer(t)
	defer server1.Stop()

	server2 := CreateTestServer(t)
	defer server2.Stop()

	server1.JoinWAN(t, server2.LANAddr)

	// create client
	consul := ClientAtServer(server1)
	consul.config.Tag = "marathon-mycluster"

	// given
	// register services in both servers
	server1.AddService(t, "serviceA", "passing", []string{"public", "marathon-mycluster"})
	server1.AddService(t, "serviceB", "passing", []string{"marathon"})
	server1.AddService(t, "serviceC", "passing", []string{"marathon"})

	server2.AddService(t, "serviceA", "passing", []string{"private", "marathon"})
	server2.AddService(t, "serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetServices("serviceA")

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Contains(t, services[0].Tags, "marathon-mycluster")

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.Name] = struct{}{}
	}
	assert.Len(t, serviceNames, 1)
	assert.Contains(t, serviceNames, "serviceA")
}

func TestGetAllServices(t *testing.T) {
	t.Parallel()
	// create cluster of 2 consul servers
	server1 := CreateTestServer(t)
	defer server1.Stop()

	server2 := CreateTestServer(t)
	defer server2.Stop()

	server1.JoinWAN(t, server2.LANAddr)

	// create client
	consul := ClientAtServer(server1)
	consul.config.Tag = "marathon"

	// given
	// register services in both servers
	server1.AddService(t, "serviceA", "passing", []string{"public", "marathon"})
	server1.AddService(t, "serviceB", "passing", []string{"marathon"})
	server1.AddService(t, "serviceC", "passing", []string{"zookeeper"})

	server2.AddService(t, "serviceA", "passing", []string{"private", "marathon"})
	server2.AddService(t, "serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetAllServices()

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 3)

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.Name] = struct{}{}
	}
	assert.Len(t, serviceNames, 2)
	assert.Contains(t, serviceNames, "serviceA")
	assert.Contains(t, serviceNames, "serviceB")
}

func TestGetServicesUsingProviderWithRetriesOnAgentFailure_ShouldRetryConfiguredNumberOfTimes(t *testing.T) {
	t.Parallel()
	server1 := CreateTestServer(t)
	defer server1.Stop()

	// create client
	consul := ClientAtServer(server1)
	consul.config.AgentFailuresTolerance = 2
	consul.config.RequestRetries = 4

	var called uint32
	provider := func(agent *consulapi.Client) ([]*service.Service, error) {
		called++
		return nil, fmt.Errorf("error")
	}

	// add failing client
	consul.AddAgent("127.0.0.2")

	// when
	consul.getServicesUsingProviderWithRetriesOnAgentFailure(provider)

	//then
	assert.Equal(t, consul.config.RequestRetries+1, called)
	assert.Len(t, consul.agents.(*ConcurrentAgents).agents, 2)
}

func TestGetServicesUsingProviderWithRetriesOnAgentFailure_ShouldRemoveFailedAgentWhenNoLocalGiven(t *testing.T) {
	t.Parallel()
	server1 := CreateTestServer(t)
	defer server1.Stop()

	// create client
	consul := ClientAtRemoteServer(server1)
	consul.config.AgentFailuresTolerance = 2
	consul.config.RequestRetries = 4

	var called uint32
	provider := func(agent *consulapi.Client) ([]*service.Service, error) {
		called++
		return nil, fmt.Errorf("error")
	}

	// add failing client
	consul.AddAgent("127.0.0.2")

	// when
	consul.getServicesUsingProviderWithRetriesOnAgentFailure(provider)

	//then
	assert.Equal(t, consul.config.RequestRetries+1, called)
	assert.Len(t, consul.agents.(*ConcurrentAgents).agents, 1)
}

func TestGetAllServices_FailingAgent_GivingUp(t *testing.T) {
	t.Parallel()
	server1 := CreateTestServer(t)
	defer server1.Stop()

	// create client
	consul := FailingClient()

	// when
	services, err := consul.GetAllServices()

	// then
	assert.EqualError(t, err, "An error occurred getting services from Consul. Giving up")
	assert.Nil(t, services)
}

func TestGetAllServices_RemovingFailingAgentsAndRetrying(t *testing.T) {
	t.Parallel()
	// create cluster of 2 consul servers
	server1 := CreateTestServer(t)
	defer server1.Stop()

	server2 := CreateTestServer(t)
	defer server2.Stop()

	server1.JoinWAN(t, server2.LANAddr)

	// create client
	consul := ClientAtServer(server1)
	consul.config.Tag = "marathon"
	consul.config.RequestRetries = 10

	// add failing clients
	for i := uint32(2); i < consul.config.RequestRetries; i++ {
		consul.AddAgent(fmt.Sprintf("127.0.0.%d", i))
	}

	// given
	// register services in both servers
	server1.AddService(t, "serviceA", "passing", []string{"public", "marathon"})
	server1.AddService(t, "serviceB", "passing", []string{"marathon"})
	server1.AddService(t, "serviceC", "passing", []string{"zookeeper"})

	server2.AddService(t, "serviceA", "passing", []string{"private", "marathon"})
	server2.AddService(t, "serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetAllServices()

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 3)

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.Name] = struct{}{}
	}
	assert.Len(t, serviceNames, 2)
	assert.Contains(t, serviceNames, "serviceA")
	assert.Contains(t, serviceNames, "serviceB")
}

func TestRegisterServices(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
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
	assert.Equal(t, "serviceA", services[0].Name)
	assert.Equal(t, []string{"marathon", "test", "marathon-task:serviceA.0"}, services[0].Tags)
}

func TestRegisterServices_CustomServiceName(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
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
	assert.Equal(t, "myCustomServiceName", services[0].Name)
}

func TestRegisterServices_MultipleRegistrations(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	app.PortDefinitions = []apps.PortDefinition{
		{
			Labels: map[string]string{"consul": "first-name", "first-tag": "tag"},
		},
		{
			Labels: map[string]string{"consul": "second-name", "second-tag": "tag"},
		},
	}
	app.Tasks[0].Host = server.Config.Bind
	app.Tasks[0].Ports = []int{8080, 8081}
	app.Labels["common-tag"] = "tag"

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.NoError(t, err)

	// when
	services, _ := consul.GetAllServices()

	// then
	assert.Len(t, services, 2)

	first, found := findServiceByName("first-name", services)
	assert.True(t, found, "first-name not found in services")
	second, found := findServiceByName("second-name", services)
	assert.True(t, found, "second-name not found in services")

	assert.Equal(t, "first-name", first.Name)
	assert.Equal(t, []string{"marathon", "first-tag", "common-tag", "marathon-task:serviceA.0"}, first.Tags)
	assert.Equal(t, "second-name", second.Name)
	assert.Equal(t, []string{"marathon", "second-tag", "common-tag", "marathon-task:serviceA.0"}, second.Tags)
}

func findServiceByName(name string, services []*service.Service) (*service.Service, bool) {
	for _, s := range services {
		if s.Name == name {
			return s, true
		}
	}
	return nil, false
}

func TestRegisterServices_InvalidHostnameShouldFail(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	app.Tasks[0].Host = "unknown.host.name.1"
	app.Labels["consul"] = ""

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.Error(t, err)
}

func TestRegisterServices_InvalidCustomServiceName(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
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
	assert.Equal(t, "serviceA", services[0].Name)
}

func TestRegisterServices_shouldReturnErrorOnFailure(t *testing.T) {
	t.Parallel()

	// given
	consul := New(Config{Port: "1234"})
	app := utils.ConsulApp("serviceA", 1)

	// when
	err := consul.Register(&app.Tasks[0], app)

	// then
	assert.Error(t, err)
}

func TestDeregisterServices(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	server.AddService(t, "serviceA", "passing", []string{"marathon"})
	server.AddService(t, "serviceB", "passing", []string{"marathon"})
	services, _ := consul.GetAllServices()
	assert.Len(t, services, 2)

	// when
	servicesA, _ := consul.GetServices("serviceA")
	consul.Deregister(servicesA[0])

	// then
	services, _ = consul.GetAllServices()
	assert.Len(t, services, 1)
	assert.Equal(t, "serviceB", services[0].Name)
}

func TestDeregisterServices_shouldReturnErrorOnFailure(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	server.AddService(t, "serviceA", "passing", []string{"marathon"})

	// when
	servicesA, _ := consul.GetServices("serviceA")
	server.Stop()
	err := consul.Deregister(servicesA[0])

	// then
	assert.Error(t, err)
}

func TestDeregisterServicesByTask(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	task := app.Tasks[0]

	server.AddService(t, "serviceA", "passing", []string{"marathon", service.MarathonTaskTag(task.ID)})
	server.AddService(t, "serviceB", "passing", []string{"marathon", service.MarathonTaskTag(apps.TaskID("other"))})
	services, _ := consul.GetAllServices()
	assert.Len(t, services, 2)

	// when
	consul.DeregisterByTask(task.ID)

	// then
	services, _ = consul.GetAllServices()
	assert.Len(t, services, 1)
	assert.Equal(t, "serviceB", services[0].Name)
}

func TestDeregisterServicesByTask_shouldReturnErrorOnFailure(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	task := app.Tasks[0]

	server.AddService(t, "serviceA", "passing", []string{"marathon", service.MarathonTaskTag(task.ID)})

	// when
	server.Stop()
	err := consul.DeregisterByTask(task.ID)

	// then
	assert.Error(t, err)
}

func TestDeregisterServicesByTask_shouldReturnNilErrorOnServiceMatchingTaskNotFound(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	task := app.Tasks[0]

	server.AddService(t, "serviceA", "passing", []string{"marathon", service.MarathonTaskTag(task.ID)})
	server.AddService(t, "serviceB", "passing", []string{"marathon", service.MarathonTaskTag(apps.TaskID("other"))})
	services, _ := consul.GetAllServices()
	assert.Len(t, services, 2)

	// when
	err := consul.DeregisterByTask("non-existing")

	// then
	assert.NoError(t, err)
	services, _ = consul.GetAllServices()
	assert.Len(t, services, 2)
}

func TestDeregisterServicesByTask_shouldDeregisterAllMatchingServicesWhenMultipleMatchGivenTaskId(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	consul := ClientAtServer(server)
	consul.config.Tag = "marathon"

	// given
	app := utils.ConsulApp("serviceA", 1)
	task := app.Tasks[0]

	server.AddService(t, "serviceA", "passing", []string{"marathon", service.MarathonTaskTag(task.ID)})
	server.AddService(t, "serviceA-bis", "passing", []string{"marathon", service.MarathonTaskTag(task.ID)})
	server.AddService(t, "serviceB", "passing", []string{"marathon", service.MarathonTaskTag(apps.TaskID("other"))})
	services, _ := consul.GetAllServices()
	assert.Len(t, services, 3)

	// when
	err := consul.DeregisterByTask(task.ID)

	// then
	assert.NoError(t, err)
	services, _ = consul.GetAllServices()
	assert.Len(t, services, 1)
}

func TestAddAgentsFromApp(t *testing.T) {
	t.Parallel()
	server := CreateTestServer(t)
	defer server.Stop()

	// create consul without any agents in cache
	consul := New(Config{
		Timeout:             timeutil.Interval{Duration: 10 * time.Millisecond},
		Port:                fmt.Sprintf("%d", server.Config.Ports.HTTP),
		ConsulNameSeparator: ".",
		Tag:                 "marathon",
	})

	// given
	app := utils.ConsulApp("serviceA", 1)
	app.Tasks[0].Host = server.Config.Bind
	server.AddService(t, "serviceA", "passing", []string{"marathon", service.MarathonTaskTag(app.Tasks[0].ID)})

	// when
	consul.AddAgentsFromApps([]*apps.App{app, utils.NonConsulApp("nonConsulApp", 1)})

	// then
	services, err := consul.GetAllServices()
	assert.Len(t, services, 1)
	assert.NoError(t, err)
}

func TestGetIgnoredHealthCheckTypes(t *testing.T) {
	t.Parallel()

	// given
	var ignoredTypes = []struct {
		config string
		parsed []string
	}{
		{",command ", []string{"COMMAND"}},
		{"tcp,http", []string{"TCP", "HTTP"}},
		{"tcp, command,", []string{"TCP", "COMMAND"}},
		{"HTTP ", []string{"HTTP"}},
		{"", []string{}},
		{" ,", []string{}},
	}

	for _, types := range ignoredTypes {
		// when
		consul := New(Config{IgnoredHealthChecks: types.config})

		// then
		assert.Equal(t, types.parsed, consul.ignoredHealthCheckTypes)
	}
}

func TestMarathonTaskToConsulServiceMapping(t *testing.T) {
	t.Parallel()

	// given
	consul := New(Config{Tag: "marathon"})
	app := &apps.App{
		ID: "someApp",
		HealthChecks: []apps.HealthCheck{
			{
				Path:                   "/api/health?with=query",
				Protocol:               "HTTP",
				Port:                   8123,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "",
				Protocol:               "HTTP",
				PortIndex:              0,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/api/health?with=query",
				Protocol:               "INVALID_PROTOCOL",
				PortIndex:              0,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/secure/health?with=query",
				Protocol:               "HTTPS",
				PortIndex:              0,
				IntervalSeconds:        50,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Protocol:               "TCP",
				PortIndex:              1,
				IntervalSeconds:        40,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Protocol:               "TCP",
				Port:                   8234,
				IntervalSeconds:        40,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Protocol: "COMMAND",
				Command: struct {
					Value string `json:"value"`
				}{Value: "echo 1"},
				IntervalSeconds:        30,
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
	services, err := consul.marathonTaskToConsulServices(task, app)
	service := services[0]

	// then
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.6", service.Address)
	assert.Equal(t, []string{"marathon", "public", "marathon-task:someTask"}, service.Tags)
	assert.Equal(t, 8090, service.Port)
	assert.Nil(t, service.Check)
	assert.Equal(t, 6, len(service.Checks))

	assert.Equal(t, consulapi.AgentServiceChecks{
		{
			HTTP:     "http://127.0.0.6:8123/api/health?with=query",
			Interval: "60s",
			Timeout:  "20s",
			Status:   "passing",
		},
		{
			HTTP:     "http://127.0.0.6:8090/",
			Interval: "60s",
			Timeout:  "20s",
			Status:   "passing",
		},
		{
			HTTP:     "https://127.0.0.6:8090/secure/health?with=query",
			Interval: "50s",
			Timeout:  "20s",
			Status:   "passing",
		},
		{
			TCP:      "127.0.0.6:8443",
			Interval: "40s",
			Timeout:  "20s",
			Status:   "passing",
		},
		{
			TCP:      "127.0.0.6:8234",
			Interval: "40s",
			Timeout:  "20s",
			Status:   "passing",
		},
		{
			Script:   "echo 1",
			Interval: "30s",
			Timeout:  "20s",
			Status:   "passing",
		},
	}, service.Checks)
}

func TestMarathonTaskToConsulServiceMapping_IgnoredHealthcheckTypes(t *testing.T) {
	t.Parallel()

	// given
	consul := New(Config{Tag: "marathon", IgnoredHealthChecks: "command,tcp"})
	app := &apps.App{
		ID: "someApp",
		HealthChecks: []apps.HealthCheck{
			{
				Path:                   "/api/health?with=query",
				Protocol:               "MESOS_HTTP",
				PortIndex:              0,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "",
				Protocol:               "HTTP",
				PortIndex:              0,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/api/health?with=query",
				Protocol:               "INVALID_PROTOCOL",
				PortIndex:              0,
				IntervalSeconds:        60,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/secure/health?with=query",
				Protocol:               "HTTPS",
				PortIndex:              0,
				IntervalSeconds:        50,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/invalid/port/index",
				Protocol:               "HTTPS",
				PortIndex:              2,
				IntervalSeconds:        50,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/negative/port/index",
				Protocol:               "HTTPS",
				PortIndex:              -1,
				IntervalSeconds:        50,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/port/out/of/range",
				Protocol:               "HTTPS",
				Port:                   65536,
				IntervalSeconds:        50,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Protocol:               "TCP",
				PortIndex:              1,
				IntervalSeconds:        40,
				TimeoutSeconds:         20,
				MaxConsecutiveFailures: 3,
			},
			{
				Protocol: "COMMAND",
				Command: struct {
					Value string `json:"value"`
				}{Value: "echo 1"},
				IntervalSeconds:        30,
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
	services, err := consul.marathonTaskToConsulServices(task, app)
	service := services[0]

	// then
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.6", service.Address)
	assert.Equal(t, []string{"marathon", "public", "marathon-task:someTask"}, service.Tags)
	assert.Equal(t, 8090, service.Port)
	assert.Nil(t, service.Check)
	assert.Equal(t, 3, len(service.Checks))

	assert.Equal(t, consulapi.AgentServiceChecks{
		{
			HTTP:     "http://127.0.0.6:8090/api/health?with=query",
			Interval: "60s",
			Timeout:  "20s",
			Status:   "passing",
		},
		{
			HTTP:     "http://127.0.0.6:8090/",
			Interval: "60s",
			Timeout:  "20s",
			Status:   "passing",
		},
		{
			HTTP:     "https://127.0.0.6:8090/secure/health?with=query",
			Interval: "50s",
			Timeout:  "20s",
			Status:   "passing",
		},
	}, service.Checks)
}

func TestMarathonTaskToConsulServiceMapping_NotResolvableTaskHost(t *testing.T) {
	t.Parallel()

	// given
	consul := New(Config{Tag: "marathon"})
	app := &apps.App{
		ID: "someApp",
		HealthChecks: []apps.HealthCheck{
			{
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
		Host:  "invalid.hostname",
		Ports: []int{8090, 8443},
	}

	// when
	_, err := consul.marathonTaskToConsulServices(task, app)

	// then
	assert.Error(t, err)
}

func Test_substituteEnvironment(t *testing.T) {
	type args struct {
		s    string
		task apps.Task
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "without port and host",
			args: args{
				s: "true",
				task: apps.Task{
					ID:    "someTask",
					Host:  "127.0.0.6",
					Ports: []int{8090, 8443},
				},
			},
			want: "true",
		},
		{
			name: "with ports and host",
			args: args{
				s: "nc $HOST $PORT0 && nc $HOST $PORT1",
				task: apps.Task{
					ID:    "someTask",
					Host:  "127.0.0.6",
					Ports: []int{8090, 8443},
				},
			},
			want: "nc 127.0.0.6 8090 && nc 127.0.0.6 8443",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := substituteEnvironment(tt.args.s, tt.args.task); got != tt.want {
				t.Errorf("substituteEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenIsUsedToConnectToConsul(t *testing.T) {
	// given
	t.Parallel()
	server := CreateSecuredTestServer(t)
	defer server.Stop()

	bareClient := ClientAtServer(server)
	bareClient.config.Tag = "marathon"

	clientWithToken := SecuredClientAtServer(server)
	clientWithToken.config.Tag = "marathon"

	// and
	app := utils.ConsulApp("serviceA", 1)
	app.Tasks[0].Host = server.Config.Bind
	app.Labels["test"] = "tag"

	// when
	err := bareClient.Register(&app.Tasks[0], app)

	// then
	assert.NoError(t, err, "Though it seems surprising, consul should not report an error here")

	// when
	services, _ := clientWithToken.GetAllServices()

	// then
	assert.Len(t, services, 0, "Registration without ACL token should be blocked by ACLs")

	// when
	err = clientWithToken.Register(&app.Tasks[0], app)
	assert.NoError(t, err, "Registering service with proper ACL token should not report errors")

	// when
	services, _ = clientWithToken.GetAllServices()

	// then
	assert.Len(t, services, 1, "Expecting a registered service after using ACL token")
	assert.Equal(t, "serviceA", services[0].Name)
	assert.Equal(t, []string{"marathon", "test", "marathon-task:serviceA.0"}, services[0].Tags)

	// when
	services, _ = bareClient.GetAllServices()

	// then
	assert.Len(t, services, 0, "Reading services list without ACL token should yield empty response")
}
