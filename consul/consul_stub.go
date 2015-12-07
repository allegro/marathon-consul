package consul

import (
	consulapi "github.com/hashicorp/consul/api"
)

type ConsulStub struct {
	services      map[string]*consulapi.AgentServiceRegistration
	ErrorServices map[string]error
}

func NewConsulStub() *ConsulStub {
	return &ConsulStub{
		services:      make(map[string]*consulapi.AgentServiceRegistration),
		ErrorServices: make(map[string]error),
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

func (c *ConsulStub) Register(service *consulapi.AgentServiceRegistration) error {
	if err, ok := c.ErrorServices[service.ID]; ok {
		return err
	} else {
		c.services[service.ID] = service
		return nil
	}
}

func (c *ConsulStub) Deregister(serviceId string, agent string) error {
	if err, ok := c.ErrorServices[serviceId]; ok {
		return err
	} else {
		delete(c.services, serviceId)
		return nil
	}
}

func (c *ConsulStub) RegisteredServicesIds() []string {
	services, _ := c.GetAllServices()
	servicesIds := []string{}
	for _, consulService := range services {
		servicesIds = append(servicesIds, consulService.ServiceID)
	}
	return servicesIds
}
