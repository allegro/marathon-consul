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
	return nil, fmt.Errorf("No Consul client available in agents cache")
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
		return nil, fmt.Errorf("Invalid agent address for Consul client")
	}
	config := consulapi.DefaultConfig()

	config.Address = fmt.Sprintf("%s:%s", host, a.config.Port)
	config.HttpClient.Timeout = a.config.Timeout

	if a.config.Token != "" {
		config.Token = a.config.Token
	}

	if a.config.SslEnabled {
		config.Scheme = "https"
	}

	if !a.config.SslVerify {
		config.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if a.config.Auth.Enabled {
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: a.config.Auth.Username,
			Password: a.config.Auth.Password,
		}
	}

	log.WithFields(log.Fields{
		"Address": config.Address,
		"Scheme": config.Scheme,
		"Timeout": config.HttpClient.Timeout,
		"BasicAuthEnabled": a.config.Auth.Enabled,
		"TokenEnabled": a.config.Token != "",
		"SslVerificationEnabled": a.config.SslVerify,
	}).Debug("Creating Consul client")

	return consulapi.NewClient(config)
}
