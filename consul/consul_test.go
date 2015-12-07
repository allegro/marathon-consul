package consul

import (
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"testing"
)

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

	// given
	// register services in both servers
	server1.AddService("serviceA", "passing", []string{"public", "marathon"})
	server1.AddService("serviceB", "passing", []string{"marathon"})
	server1.AddService("serviceC", "passing", []string{"zookeeper"})

	server2.AddService("serviceA", "passing", []string{"private", "marathon"})
	server2.AddService("serviceB", "passing", []string{"zookeeper"})

	// when
	services, err := consul.GetAllServices()
	if err != nil {
		t.Fatal("Could not get services from consul")
	}

	// then
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

	// given
	service := &consulapi.AgentServiceRegistration{
		Name:    "serviceA",
		Address: "127.0.0.1",
		Port:    8080,
		Tags:    []string{"test", "marathon"},
		Check: &consulapi.AgentServiceCheck{
			HTTP:     "http://127.0.0.1:8080/status/ping",
			Interval: "60s",
		},
	}

	// when
	consul.Register(service)

	// then
	services, _ := consul.GetAllServices()
	assert.Len(t, services, 1)
	assert.Equal(t, "serviceA", services[0].ServiceName)
	assert.Equal(t, []string{"test", "marathon"}, services[0].ServiceTags)
}

func TestDeregisterServices(t *testing.T) {
	t.Parallel()
	server := CreateConsulTestServer("dc1", t)
	defer server.Stop()

	consul := ConsulClientAtServer(server)

	// given
	server.AddService("serviceA", "passing", []string{"marathon"})
	server.AddService("serviceB", "passing", []string{"marathon"})
	services, _ := consul.GetAllServices()
	assert.Len(t, services, 2)

	// when
	consul.Deregister("serviceA", server.Config.NodeName)

	// then
	services, _ = consul.GetAllServices()
	assert.Len(t, services, 1)
	assert.Equal(t, "serviceB", services[0].ServiceName)
}
