package consul

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
)

type Agents interface {
	GetAgent(agentAddress string) (agent *consulapi.Client, err error)
	GetAnyAgent() (agent *Agent, err error)
	RemoveAgent(agentAddress string)
}

type ConcurrentAgents struct {
	agents map[string]*Agent
	config *ConsulConfig
	lock   sync.Mutex
	client *http.Client
}

func NewAgents(config *ConsulConfig) *ConcurrentAgents {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !config.SslVerify,
			},
		},
		Timeout: config.Timeout,
	}
	return &ConcurrentAgents{
		agents: make(map[string]*Agent),
		config: config,
		client: client,
	}
}

func (a *ConcurrentAgents) GetAnyAgent() (*Agent, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if len(a.agents) > 0 {
		ipAddress := a.getRandomAgentIpAddress()
		return a.agents[ipAddress], nil
	}
	return nil, fmt.Errorf("No Consul client available in agents cache")
}

func (a *ConcurrentAgents) getRandomAgentIpAddress() string {
	ipAddresses := []string{}
	for ipAddress := range a.agents {
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
		return agent.Client, nil
	}

	newAgent, err := a.createAgent(ipAddress)
	if err != nil {
		return nil, err
	}
	a.addAgent(ipAddress, newAgent)

	return newAgent.Client, nil
}

func (a *ConcurrentAgents) addAgent(agentHost string, agent *Agent) {
	a.agents[agentHost] = agent
	a.updateAgentsCacheSizeMetricValue()
}

func (a *ConcurrentAgents) updateAgentsCacheSizeMetricValue() {
	metrics.UpdateGauge("consul.agents.cache.size", int64(len(a.agents)))
}
