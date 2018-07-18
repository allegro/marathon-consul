package consul

import (
	"fmt"
	"net"
	"testing"
	"time"

	timeutil "github.com/allegro/marathon-consul/time"
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
	ports, err := getPorts(5)
	assert.NoError(t, err)

	return &testutil.TestPortConfig{
		DNS:     ports[0],
		HTTP:    ports[1],
		SerfLan: ports[2],
		SerfWan: ports[3],
		Server:  ports[4],
	}
}

// Ask the kernel for free open ports that are ready to use
func getPorts(number int) ([]int, error) {
	ports := make([]int, number)
	listener := make([]*net.TCPListener, number)
	defer func() {
		for _, l := range listener {
			l.Close()
		}

	}()
	for i := 0; i < number; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return nil, err
		}

		listener[i], err = net.ListenTCP("tcp", addr)
		if err != nil {
			return nil, err
		}
		ports[i] = listener[i].Addr().(*net.TCPAddr).Port
	}
	return ports, nil
}

func ClientAtServer(server *testutil.TestServer) *Consul {
	return clientAtServer(server, true)
}

func ClientAtRemoteServer(server *testutil.TestServer) *Consul {
	return clientAtServer(server, false)
}

func clientAtServer(server *testutil.TestServer, local bool) *Consul {
	return consulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP, local)
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

func consulClientAtAddress(host string, port int, local bool) *Consul {
	localAgent := ""
	if local {
		localAgent = host
	}
	config := Config{
		Timeout:             timeutil.Interval{Duration: 10 * time.Second},
		Port:                fmt.Sprintf("%d", port),
		ConsulNameSeparator: ".",
		EnableTagOverride:   true,
		LocalAgentHost:      localAgent,
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
