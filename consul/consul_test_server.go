package consul

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/testutil"
)

var dcOffset uint32 = 1
var failingClientOffset uint32 = 1

func CreateConsulTestServer(t *testing.T) *testutil.TestServer {
	return testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Datacenter = fmt.Sprint("dc-", atomic.AddUint32(&dcOffset, 1))
	})
}

func ConsulClientAtServer(server *testutil.TestServer) *Consul {
	return consulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP)
}

func FailingConsulClient() *Consul {
	return consulClientAtAddress(fmt.Sprint("127.5.5.", atomic.AddUint32(&failingClientOffset, 1)), 5555)
}

func consulClientAtAddress(host string, port int) *Consul {
	config := ConsulConfig{
		Timeout:             10 * time.Millisecond,
		Port:                fmt.Sprintf("%d", port),
		ConsulNameSeparator: ".",
	}
	consul := New(config)
	// initialize the agents cache with a single client pointing at provided location
	consul.GetAgent(host)
	return consul
}
