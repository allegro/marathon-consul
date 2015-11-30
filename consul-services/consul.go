package consul

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/CiscoCloud/mesos-consul/registry"

	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

type Consul struct {
	agents map[string]*consulapi.Client
	config consulConfig
}

//
func New() *Consul {
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

	config.Address = fmt.Sprintf("%s:%s", address, c.config.port)
	log.Debugf("consul address: %s", config.Address)

	if c.config.token != "" {
		log.Debugf("setting token to %s", c.config.token)
		config.Token = c.config.token
	}

	if c.config.sslEnabled {
		log.Debugf("enabling SSL")
		config.Scheme = "https"
	}

	if !c.config.sslVerify {
		log.Debugf("disabled SSL verification")
		config.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if c.config.auth.Enabled {
		log.Debugf("setting basic auth")
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: c.config.auth.Username,
			Password: c.config.auth.Password,
		}
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		log.Fatal("consul: ", address)
	}
	return client
}

func (c *Consul) GetAllServices() ([]*consulapi.CatalogService, error) {
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

func (c *Consul) Register(service *registry.Service) {
	if _, ok := c.agents[service.Agent]; !ok {
		// Agent connection not saved. Connect.
		c.agents[service.Agent] = c.newAgent(service.Agent)
	}

	log.WithFields(log.Fields{
		"Name": service.Name,
		"Id":   service.ID,
		"Tags": service.Tags,
		"Host": service.Address,
		"Port": service.Port,
	}).Info("Registering")

	s := &consulapi.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Name,
		Port:    service.Port,
		Address: service.Address,
		Check: &consulapi.AgentServiceCheck{
			TTL:      service.Check.TTL,
			Script:   service.Check.Script,
			HTTP:     service.Check.HTTP,
			Interval: service.Check.Interval,
		},
	}

	if len(service.Tags) > 0 {
		s.Tags = service.Tags
	}

	err := c.agents[service.Agent].Agent().ServiceRegister(s)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"Name": s.Name,
			"Id":   s.ID,
			"Tags": s.Tags,
			"Host": s.Address,
			"Port": s.Port,
		}).Warnf("Unable to register")
		return
	}
}

func (c *Consul) Deregister(serviceId string, agent string) {
	if _, ok := c.agents[agent]; !ok {
		// Agent connection not saved. Connect.
		c.agents[agent] = c.newAgent(agent)
	}

	log.WithField("Id", serviceId).Info("Deregistering")

	err := c.agents[agent].Agent().ServiceDeregister(serviceId)
	if err != nil {
		log.WithError(err).WithField("Id", serviceId).Info("Deregistering")
		return
	}
}

func (c *Consul) deregister(agent string, service *consulapi.AgentServiceRegistration) error {
	if _, ok := c.agents[agent]; !ok {
		// Agent connection not saved. Connect.
		c.agents[agent] = c.newAgent(agent)
	}

	return c.agents[agent].Agent().ServiceDeregister(service.ID)
}
