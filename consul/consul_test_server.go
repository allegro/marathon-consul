package consul

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul/testutil"
)

func CreateConsulTestServer(dc string, t *testing.T) *testutil.TestServer {
	return testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Datacenter = dc
	})
}

func ConsulClientAtServer(server *testutil.TestServer) *Consul {
	return consulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP)
}

func consulClientAtAddress(host string, port int) *Consul {
	config := ConsulConfig{
		Timeout: 10 * time.Millisecond,
		Port:    fmt.Sprintf("%d", port),
	}
	consul := New(config)
	// initialize the agents cache with a single client pointing at provided location
	consul.GetAgent(host)
	return consul
}
