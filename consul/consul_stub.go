package consul

import (
	consulapi "github.com/hashicorp/consul/api"
)

type ConsulStub struct {
	services map[string]*consulapi.AgentServiceRegistration
}

func NewConsulStub() *ConsulStub {
	return &ConsulStub{
		services: make(map[string]*consulapi.AgentServiceRegistration),
	}
}

func (c ConsulStub) GetAllServices() ([]*consulapi.CatalogService, error) {
	var catalog []*consulapi.CatalogService
	for _, s := range c.services {
		catalog = append(catalog, &consulapi.CatalogService{
			Address:        s.Address,
			ServiceAddress: s.Address,
			ServicePort:    s.Port,
			ServiceTags:    s.Tags,
			ServiceID:      s.ID,
			ServiceName:    s.Name,
		})
	}
	return catalog, nil
}

func (c *ConsulStub) Register(service *consulapi.AgentServiceRegistration) {
	c.services[service.ID] = service
}

func (c *ConsulStub) Deregister(serviceId string, agent string) {
	delete(c.services, serviceId)
}
