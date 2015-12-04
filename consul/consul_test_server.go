package consul

import (
	"fmt"
	"github.com/hashicorp/consul/testutil"
	"testing"
)

func CreateConsulTestServer(dc string, t *testing.T) *testutil.TestServer {
	return testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Datacenter = dc
	})
}

func CreateNamedConsulTestServer(hostname string, dc string, t *testing.T) *testutil.TestServer {
	return testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Datacenter = dc
		c.NodeName = hostname
	})
}

func ConsulClientAtServer(server *testutil.TestServer) *Consul {
	return consulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP)
}

func consulClientAtAddress(host string, port int) *Consul {
	config := ConsulConfig{
		Port: fmt.Sprintf("%d", port),
	}
	consul := New(config)
	// initialize the agents cache with a single client pointing at provided location
	consul.agents.GetAgent(host)
	return consul
}
