package consul

import (
	"crypto/tls"
	"fmt"
	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"math/rand"
	"net/http"
	"sync"
)

type Agents interface {
	GetAgent(string) (*consulapi.Client, error)
	GetAnyAgent() (*consulapi.Client, error)
}

type ConcurrentAgents struct {
	agents map[string]*consulapi.Client
	config *ConsulConfig
	lock   sync.Mutex
}

func NewAgents(config *ConsulConfig) *ConcurrentAgents {
	return &ConcurrentAgents{
		agents: make(map[string]*consulapi.Client),
		config: config,
	}
}

func (a *ConcurrentAgents) GetAnyAgent() (*consulapi.Client, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if len(a.agents) > 0 {
		return a.agents[a.getRandomAgentHost()], nil
	}
	return nil, fmt.Errorf("No agent available")
}

func (a *ConcurrentAgents) getRandomAgentHost() string {
	hosts := []string{}
	for host, _ := range a.agents {
		hosts = append(hosts, host)
	}
	idx := rand.Intn(len(a.agents))
	return hosts[idx]
}

func (a *ConcurrentAgents) GetAgent(agentHost string) (*consulapi.Client, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if agent, ok := a.agents[agentHost]; ok {
		return agent, nil
	}

	newAgent, err := a.createAgent(agentHost)
	if err != nil {
		return nil, err
	}
	a.addAgent(agentHost, newAgent)
	return newAgent, nil
}

func (a *ConcurrentAgents) addAgent(agentHost string, agent *consulapi.Client) {
	a.agents[agentHost] = agent
}

func (a *ConcurrentAgents) createAgent(host string) (*consulapi.Client, error) {
	if host == "" {
		return nil, fmt.Errorf("Invalid address for Agent")
	}
	config := consulapi.DefaultConfig()

	config.Address = fmt.Sprintf("%s:%s", host, a.config.Port)
	log.Debugf("Consul address: %s", config.Address)

	if a.config.Token != "" {
		log.Debugf("Setting token to %s", a.config.Token)
		config.Token = a.config.Token
	}

	if a.config.SslEnabled {
		log.Debugf("Enabling SSL")
		config.Scheme = "https"
	}

	log.Debugf("Setting timeout to %s", a.config.Timeout.String())
	config.HttpClient.Timeout = a.config.Timeout

	if !a.config.SslVerify {
		log.Debugf("Disabled SSL verification")
		config.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if a.config.Auth.Enabled {
		log.Debugf("Setting basic auth")
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: a.config.Auth.Username,
			Password: a.config.Auth.Password,
		}
	}

	return consulapi.NewClient(config)
}
