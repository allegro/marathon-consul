package consul

import (
	"crypto/tls"
	"fmt"
	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
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

	for _, agent := range a.agents {
		return agent, nil
	}
	return nil, fmt.Errorf("No agent available")
}

func (a *ConcurrentAgents) GetAgent(agentAddress string) (*consulapi.Client, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if agent, ok := a.agents[agentAddress]; ok {
		return agent, nil
	}

	newAgent, err := a.createAgent(agentAddress)
	if err != nil {
		return nil, err
	}
	a.addAgent(agentAddress, newAgent)
	return newAgent, nil
}

func (a *ConcurrentAgents) addAgent(agentAddress string, agent *consulapi.Client) {
	a.agents[agentAddress] = agent
}

func (a *ConcurrentAgents) createAgent(address string) (*consulapi.Client, error) {
	if address == "" {
		return nil, fmt.Errorf("Invalid addres for Agent")
	}
	config := consulapi.DefaultConfig()

	config.Address = fmt.Sprintf("%s:%s", address, a.config.Port)
	log.Debugf("consul address: %s", config.Address)

	if a.config.Token != "" {
		log.Debugf("setting token to %s", a.config.Token)
		config.Token = a.config.Token
	}

	if a.config.SslEnabled {
		log.Debugf("enabling SSL")
		config.Scheme = "https"
	}

	if !a.config.SslVerify {
		log.Debugf("disabled SSL verification")
		config.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if a.config.Auth.Enabled {
		log.Debugf("setting basic auth")
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: a.config.Auth.Username,
			Password: a.config.Auth.Password,
		}
	}

	return consulapi.NewClient(config)
}
