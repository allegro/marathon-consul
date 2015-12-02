package consul

import (
	"crypto/tls"
	"fmt"
	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"net/http"
	"github.com/CiscoCloud/marathon-consul/metrics"
)

type ConsulServices interface {
	GetAllServices() ([]*consulapi.CatalogService, error)
	Register(service *consulapi.AgentServiceRegistration)
	Deregister(serviceId string, agent string)
}

type Consul struct {
	agents map[string]*consulapi.Client
	config ConsulConfig
}

func New(config ConsulConfig) *Consul {
	return &Consul{
		agents: make(map[string]*consulapi.Client),
		config: config,
	}
}

// client()
//   Return a consul client at the specified address
func (c *Consul) client(address string) *consulapi.Client {
	if address == "" {
		log.Warn("No address to Consul.Agent")
		return nil
	}

	if _, ok := c.agents[address]; !ok {
		// Agent connection not saved. Connect.
		c.agents[address] = c.newAgent(address)
	}

	return c.agents[address]
}

// newAgent()
//   Connect to a new agent specified by address
//
func (c *Consul) newAgent(address string) *consulapi.Client {
	if address == "" {
		log.Warnf("No address to Consul.NewAgent")
		return nil
	}

	config := consulapi.DefaultConfig()

	config.Address = fmt.Sprintf("%s:%s", address, c.config.Port)
	log.Debugf("consul address: %s", config.Address)

	if c.config.Token != "" {
		log.Debugf("setting token to %s", c.config.Token)
		config.Token = c.config.Token
	}

	if c.config.SslEnabled {
		log.Debugf("enabling SSL")
		config.Scheme = "https"
	}

	if !c.config.SslVerify {
		log.Debugf("disabled SSL verification")
		config.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if c.config.Auth.Enabled {
		log.Debugf("setting basic auth")
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: c.config.Auth.Username,
			Password: c.config.Auth.Password,
		}
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		log.Fatal("consul: ", address)
	}
	return client
}

func (c *Consul) GetAllServices() ([]*consulapi.CatalogService, error) {
	// TODO: first returned agent might already be unavailable (slave failure etc.), should retry with another
	agent, err := c.GetAnyAgent()
	if err != nil {
		return nil, err
	}
	datacenters, err := agent.Catalog().Datacenters()
	if err != nil {
		return nil, err
	}
	var allInstances []*consulapi.CatalogService

	for _, dc := range datacenters {
		dcAwareQuery := &consulapi.QueryOptions{
			Datacenter: dc,
		}
		services, _, err := agent.Catalog().Services(dcAwareQuery)
		if err != nil {
			return nil, err
		}
		for service, tags := range services {
			if contains(tags, "marathon") {
				serviceInstances, _, err := agent.Catalog().Service(service, "marathon", dcAwareQuery)
				if err != nil {
					return nil, err
				}
				allInstances = append(allInstances, serviceInstances...)
			}
		}
	}
	return allInstances, nil
}

func contains(slice []string, search string) bool {
	for _, element := range slice {
		if element == search {
			return true
		}
	}
	return false
}

func (c *Consul) GetAnyAgent() (*consulapi.Client, error) {
	for _, agent := range c.agents {
		return agent, nil
	}
	return nil, fmt.Errorf("No consul agent available")
}

func (c *Consul) Register(service *consulapi.AgentServiceRegistration) {
	metrics.Time("consul.register", func() { c.register(service) })
}

func (c *Consul) register(service *consulapi.AgentServiceRegistration) error {
	if _, ok := c.agents[service.Address]; !ok {
		// Agent connection not saved. Connect.
		c.agents[service.Address] = c.newAgent(service.Address)
	}

	log.WithFields(log.Fields{
		"Name": service.Name,
		"Id":   service.ID,
		"Tags": service.Tags,
		"Host": service.Address,
		"Port": service.Port,
	}).Info("Registering")

	err := c.agents[service.Address].Agent().ServiceRegister(service)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"Name": service.Name,
			"Id":   service.ID,
			"Tags": service.Tags,
			"Host": service.Address,
			"Port": service.Port,
		}).Warnf("Unable to register")
	}
	return err
}

func (c *Consul) Deregister(serviceId string, agent string) {
	metrics.Time("consul.deregister", func() { c.deregister(serviceId, agent) })
}

func (c *Consul) deregister(serviceId string, agent string) error {
	if _, ok := c.agents[agent]; !ok {
		// Agent connection not saved. Connect.
		c.agents[agent] = c.newAgent(agent)
	}

	log.WithField("Id", serviceId).Info("Deregistering")

	err := c.agents[agent].Agent().ServiceDeregister(serviceId)
	if err != nil {
		log.WithError(err).WithField("Id", serviceId).Info("Deregistering")
	}
	return err
}
