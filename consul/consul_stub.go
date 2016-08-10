package consul

import (
	"github.com/allegro/marathon-consul/apps"
	consulapi "github.com/hashicorp/consul/api"
)

type Stub struct {
	services         map[apps.TaskID]*consulapi.AgentServiceRegistration
	ErrorServices    map[apps.TaskID]error
	ErrorGetServices map[string]error
	consul           *Consul
}

func NewConsulStub() *Stub {
	return NewConsulStubWithTag("marathon")
}

func NewConsulStubWithTag(tag string) *Stub {
	return &Stub{
		services:         make(map[apps.TaskID]*consulapi.AgentServiceRegistration),
		ErrorServices:    make(map[apps.TaskID]error),
		ErrorGetServices: make(map[string]error),
		consul:           New(Config{Tag: tag, ConsulNameSeparator: "."}),
	}
}

func (c Stub) GetAllServices() ([]*consulapi.CatalogService, error) {
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

func (c Stub) GetServices(name string) ([]*consulapi.CatalogService, error) {
	if err, ok := c.ErrorGetServices[name]; ok {
		return nil, err
	}
	var catalog []*consulapi.CatalogService
	for _, s := range c.services {
		if s.Name == name && contains(s.Tags, c.consul.config.Tag) {
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

func (c *Stub) Register(task *apps.Task, app *apps.App) error {
	if err, ok := c.ErrorServices[task.ID]; ok {
		return err
	}
	service, err := c.consul.marathonTaskToConsulService(task, app)
	if err != nil {
		return err
	}
	c.services[task.ID] = service
	return nil
}

func (c *Stub) ServiceName(app *apps.App) string {
	return c.consul.ServiceName(app)
}

func (c *Stub) Deregister(serviceID apps.TaskID, agent string) error {
	if err, ok := c.ErrorServices[serviceID]; ok {
		return err
	}
	delete(c.services, serviceID)
	return nil
}

func (c *Stub) RegisteredServicesIds() []string {
	services, _ := c.GetAllServices()
	servicesIds := []string{}
	for _, consulService := range services {
		servicesIds = append(servicesIds, consulService.ServiceID)
	}
	return servicesIds
}

func (c *Stub) GetAgent(agentAddress string) (*consulapi.Client, error) {
	return nil, nil
}
