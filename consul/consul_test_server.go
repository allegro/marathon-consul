package consul

import (
	"fmt"
	"testing"
	"time"

	timeutil "github.com/allegro/marathon-consul/time"
	"github.com/hashicorp/consul/test/porter"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
)

func CreateTestServer(t *testing.T) *testutil.TestServer {
	server, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.Datacenter = fmt.Sprint("dc-", time.Now().UnixNano())
		c.Ports = testPortConfig(t)
	})

	assert.NoError(t, err)

	return server
}

const MasterToken = "masterToken"

func CreateSecuredTestServer(t *testing.T) *testutil.TestServer {
	server, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.Datacenter = fmt.Sprint("dc-", time.Now().UnixNano())
		c.Ports = testPortConfig(t)
		c.ACLDatacenter = c.Datacenter
		c.ACLDefaultPolicy = "deny"
		c.ACLMasterToken = MasterToken
	})

	assert.NoError(t, err)

	return server
}
func testPortConfig(t *testing.T) *testutil.TestPortConfig {
	ports, err := porter.RandomPorts(6)
	assert.NoError(t, err)

	return &testutil.TestPortConfig{
		DNS:     ports[0],
		HTTP:    ports[1],
		HTTPS:   ports[2],
		SerfLan: ports[3],
		SerfWan: ports[4],
		Server:  ports[5],
	}
}

func ClientAtServer(server *testutil.TestServer) *Consul {
	return consulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP)
}

func SecuredClientAtServer(server *testutil.TestServer) *Consul {
	return secureConsulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP)
}

func FailingClient() *Consul {
	host, port := "192.0.2.5", 5555
	config := Config{
		Port:                fmt.Sprintf("%d", port),
		ConsulNameSeparator: ".",
		EnableTagOverride:   true,
	}
	consul := New(config)
	// initialize the agents cache with a single client pointing at provided location
	consul.AddAgent(host)
	return consul
}

func consulClientAtAddress(host string, port int) *Consul {
	config := Config{
		Timeout:             timeutil.Interval{Duration: 10 * time.Second},
		Port:                fmt.Sprintf("%d", port),
		ConsulNameSeparator: ".",
		EnableTagOverride:   true,
		LocalAgentHost:      host,
	}
	consul := New(config)
	// initialize the agents cache with a single client pointing at provided location
	consul.AddAgent(host)
	return consul
}

func secureConsulClientAtAddress(host string, port int) *Consul {
	config := Config{
		Timeout:             timeutil.Interval{Duration: 10 * time.Second},
		Port:                fmt.Sprintf("%d", port),
		ConsulNameSeparator: ".",
		EnableTagOverride:   true,
		LocalAgentHost:      host,
		Token:               MasterToken,
	}
	consul := New(config)
	// initialize the agents cache with a single client pointing at provided location
	consul.AddAgent(host)
	return consul
}
