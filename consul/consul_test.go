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
	consul, _ := ConsulClientAtServer(server1)

	// given
	// register services in both servers
	server1.AddService("serviceA", "passing", []string{"public", "marathon"})
	server1.AddService("serviceB", "passing", []string{"marathon"})
	server1.AddService("serviceC", "passing", []string{"zookeeper"})

	server2.AddService("serviceA", "passing", []string{"private", "marathon"})
	server2.AddService("serviceB", "passing", []string{"zookeeper"})

	// when
	services, _ := consul.GetAllServices()

	// then
	assert.Equal(t, 3, len(services))

	serviceNames := make(map[string]struct{})
	for _, s := range services {
		serviceNames[s.ServiceName] = struct{}{}
	}
	assert.Equal(t, 2, len(serviceNames))
	assert.Contains(t, serviceNames, "serviceA")
	assert.Contains(t, serviceNames, "serviceB")
}

func TestRegisterServices(t *testing.T) {
	t.Parallel()
	server := CreateConsulTestServer("dc1", t)
	defer server.Stop()

	consul, _ := ConsulClientAtServer(server)

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
	assert.Equal(t, 1, len(services))
	assert.Equal(t, "serviceA", services[0].ServiceName)
	assert.Equal(t, []string{"test", "marathon"}, services[0].ServiceTags)
}
