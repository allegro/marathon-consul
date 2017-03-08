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
	ports, err := getPorts(6)
	assert.NoError(t, err)
	return testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Datacenter = fmt.Sprint("dc-", time.Now().UnixNano())
		c.Ports = &testutil.TestPortConfig{
			DNS:     ports[0],
			HTTP:    ports[1],
			RPC:     ports[2],
			SerfLan: ports[3],
			SerfWan: ports[4],
			Server:  ports[5],
		}
		c.DisableCheckpoint = true
		c.Performance.RaftMultiplier = 1
	})
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
	return consulClientAtAddress(server.Config.Bind, server.Config.Ports.HTTP)
}

func FailingClient() *Consul {
	return consulClientAtAddress("127.5.5.5", 5555)
}

func consulClientAtAddress(host string, port int) *Consul {
	config := Config{
		Timeout:             timeutil.Interval{Duration: 10 * time.Second},
		Port:                fmt.Sprintf("%d", port),
		ConsulNameSeparator: ".",
	}
	consul := New(config)
	// initialize the agents cache with a single client pointing at provided location
	consul.AddAgent(host)
	return consul
}
