package consul

import (
	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/metrics"
	consulapi "github.com/hashicorp/consul/api"
)

type ConsulServices interface {
	GetAllServices() ([]*consulapi.CatalogService, error)
	Register(service *consulapi.AgentServiceRegistration) error
	Deregister(serviceId string, agent string) error
}

type Consul struct {
	agents Agents
}

func New(config ConsulConfig) *Consul {
	return &Consul{
		agents: NewAgents(&config),
	}
}

func (c *Consul) GetAllServices() ([]*consulapi.CatalogService, error) {
	// TODO: first returned agent might already be unavailable (slave failure etc.), should retry with another
	agent, err := c.agents.GetAnyAgent()
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

func (c *Consul) Register(service *consulapi.AgentServiceRegistration) error {
	var err error
	metrics.Time("consul.register", func() { err = c.register(service) })
	return err
}

func (c *Consul) register(service *consulapi.AgentServiceRegistration) error {
	agent, err := c.agents.GetAgent(service.Address)
	if err != nil {
		return err
	}
	fields := log.Fields{
		"Name": service.Name,
		"Id":   service.ID,
		"Tags": service.Tags,
		"Host": service.Address,
		"Port": service.Port,
	}
	log.WithFields(fields).Info("Registering")

	err = agent.Agent().ServiceRegister(service)
	if err != nil {
		log.WithError(err).WithFields(fields).Error("Unable to register")
	}
	return err
}

func (c *Consul) Deregister(serviceId string, agentHost string) error {
	var err error
	metrics.Time("consul.deregister", func() { err = c.deregister(serviceId, agentHost) })
	return err
}

func (c *Consul) deregister(serviceId string, agentHost string) error {
	agent, err := c.agents.GetAgent(agentHost)
	if err != nil {
		return err
	}

	log.WithField("Id", serviceId).WithField("Host", agentHost).Info("Deregistering")

	err = agent.Agent().ServiceDeregister(serviceId)
	if err != nil {
		log.WithError(err).WithField("Id", serviceId).WithField("Host", agentHost).Error("Unable to deregister")
	}
	return err
}
