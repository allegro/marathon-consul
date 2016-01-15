package consul

import (
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/tasks"
	consulapi "github.com/hashicorp/consul/api"
)

type ConsulStub struct {
	services      map[tasks.Id]*consulapi.AgentServiceRegistration
	ErrorServices map[tasks.Id]error
	consul        *Consul
}

func NewConsulStub() *ConsulStub {
	return NewConsulStubWithTag("marathon")
}

func NewConsulStubWithTag(tag string) *ConsulStub {
	return &ConsulStub{
		services:      make(map[tasks.Id]*consulapi.AgentServiceRegistration),
		ErrorServices: make(map[tasks.Id]error),
		consul:        New(ConsulConfig{Tag: tag}),
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

func (c ConsulStub) GetServices(name tasks.AppId) ([]*consulapi.CatalogService, error) {
	var catalog []*consulapi.CatalogService
	for _, s := range c.services {
		if s.Name == name.ConsulServiceName() && contains(s.Tags, c.consul.config.Tag) {
			catalog = append(catalog, &consulapi.CatalogService{
				Address:        s.Address,
				ServiceAddress: s.Address,
				ServicePort:    s.Port,
				ServiceTags:    s.Tags,
				ServiceID:      s.ID,
				ServiceName:    s.Name,
			})
		}
	}
	return catalog, nil
}

func (c *ConsulStub) Register(task *tasks.Task, app *apps.App) error {
	if err, ok := c.ErrorServices[task.ID]; ok {
		return err
	} else {
		c.services[task.ID] = c.consul.marathonTaskToConsulService(task, app.HealthChecks, app.Labels)
		return nil
	}
}

func (c *ConsulStub) Deregister(serviceId tasks.Id, agent string) error {
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

func (c *ConsulStub) GetAgent(agentAddress string) (*consulapi.Client, error) {
	return nil, nil
}
