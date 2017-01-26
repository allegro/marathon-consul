package consul

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
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
	config *Config
	lock   sync.Mutex
	client *http.Client
}

func NewAgents(config *Config) *ConcurrentAgents {
	tlsClientConfig := &tls.Config{
		InsecureSkipVerify: !config.SslVerify,
	}

	if config.SslEnabled {

		if config.SslCert != "" {
			log.Debug("Enabling SSL cert")

			cert, err := tls.LoadX509KeyPair(config.SslCert, config.SslCert)
			if err != nil {
				log.Fatal(err)
			}

			tlsClientConfig.Certificates = []tls.Certificate{cert}
		}

		if config.SslCaCert != "" {
			log.Debug("Enabling SSL CA certs")

			caCert, err := ioutil.ReadFile(config.SslCaCert)
			if err != nil {
				log.Fatal(err)
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			tlsClientConfig.RootCAs = caCertPool
		}

	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: tlsClientConfig,
		},
		Timeout: config.Timeout.Duration,
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
		ipAddress := a.getRandomAgentIPAddress()
		return a.agents[ipAddress], nil
	}
	return nil, errors.New("No Consul client available in agents cache")
}

func (a *ConcurrentAgents) getRandomAgentIPAddress() string {
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
