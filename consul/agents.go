package consul

import (
	"crypto/tls"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
	"math/rand"
	"net/http"
	"sync"
)

type Agents interface {
	GetAgent(agentAddress string) (agent *consulapi.Client, err error)
	GetAnyAgent() (agent *consulapi.Client, ipAddress string, err error)
	RemoveAgent(agentAddress string)
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

func (a *ConcurrentAgents) GetAnyAgent() (*consulapi.Client, string, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if len(a.agents) > 0 {
		ipAddress := a.getRandomAgentIpAddress()
		return a.agents[ipAddress], ipAddress, nil
	}
	return nil, "", fmt.Errorf("No Consul client available in agents cache")
}

func (a *ConcurrentAgents) getRandomAgentIpAddress() string {
	ipAddresses := []string{}
	for ipAddress, _ := range a.agents {
		ipAddresses = append(ipAddresses, ipAddress)
	}
	idx := rand.Intn(len(a.agents))
	return ipAddresses[idx]
}

func (a *ConcurrentAgents) RemoveAgent(agentAddress string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if IP, err := utils.HostToIPv4(agentAddress); err != nil {
		log.WithError(err).Error("Could not remove agent from cache")
	} else {
		ipAddress := IP.String()
		log.WithField("Address", ipAddress).Info("Removing agent from cache")
		delete(a.agents, ipAddress)
		a.updateAgentsCacheSizeMetricValue()
	}
}

func (a *ConcurrentAgents) GetAgent(agentAddress string) (*consulapi.Client, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	IP, err := utils.HostToIPv4(agentAddress)
	if err != nil {
		return nil, err
	}
	ipAddress := IP.String()

	if agent, ok := a.agents[ipAddress]; ok {
		return agent, nil
	}

	newAgent, err := a.createAgent(ipAddress)
	if err != nil {
		return nil, err
	}
	a.addAgent(ipAddress, newAgent)

	return newAgent, nil
}

func (a *ConcurrentAgents) addAgent(agentHost string, agent *consulapi.Client) {
	a.agents[agentHost] = agent
	a.updateAgentsCacheSizeMetricValue()
}

func (a *ConcurrentAgents) createAgent(ipAddress string) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()

	config.Address = fmt.Sprintf("%s:%s", ipAddress, a.config.Port)
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
		"Address":                config.Address,
		"Scheme":                 config.Scheme,
		"Timeout":                config.HttpClient.Timeout,
		"BasicAuthEnabled":       a.config.Auth.Enabled,
		"TokenEnabled":           a.config.Token != "",
		"SslVerificationEnabled": a.config.SslVerify,
	}).Debug("Creating Consul client")

	return consulapi.NewClient(config)
}

func (a *ConcurrentAgents) updateAgentsCacheSizeMetricValue() {
	metrics.UpdateGauge("consul.agents.cache.size", int64(len(a.agents)))
}
