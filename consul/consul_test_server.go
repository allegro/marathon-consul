package consul

import (
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"testing"
)

func CreateConsulTestServer(dc string, t *testing.T) *testutil.TestServer {
	return testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Datacenter = dc
	})
}

func ConsulClientAtServer(server *testutil.TestServer) (*Consul, error) {
	return consulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP)
}

func consulClientAtAddress(host string, port int) (*Consul, error) {
	config := consulapi.DefaultConfig()
	config.Address = fmt.Sprintf("%s:%d", host, port)
	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	consul := New(ConsulConfig{})
	consul.agents[host] = client
	return consul, nil
}
