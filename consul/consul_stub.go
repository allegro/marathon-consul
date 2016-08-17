package consul

import (
	"fmt"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/service"
	consulapi "github.com/hashicorp/consul/api"
)

// TODO this should be a service registry stub in the service package, requires abstracting from AgentServiceRegistration
type ConsulStub struct {
	services                   map[service.ServiceId]*consulapi.AgentServiceRegistration
	failGetServicesForNames    map[string]bool
	failRegisterForIds         map[apps.TaskId]bool
	failDeregisterByTaskForIds map[apps.TaskId]bool
	failDeregisterForIds       map[service.ServiceId]bool
	consul                     *Consul
}

func NewConsulStub() *ConsulStub {
	return NewConsulStubWithTag("marathon")
}

func NewConsulStubWithTag(tag string) *ConsulStub {
	return &ConsulStub{
		services:                   make(map[service.ServiceId]*consulapi.AgentServiceRegistration),
		failGetServicesForNames:    make(map[string]bool),
		failRegisterForIds:         make(map[apps.TaskId]bool),
		failDeregisterByTaskForIds: make(map[apps.TaskId]bool),
		failDeregisterForIds:       make(map[service.ServiceId]bool),
		consul:                     New(ConsulConfig{Tag: tag, ConsulNameSeparator: "."}),
	}
}

func (c ConsulStub) GetAllServices() ([]*service.Service, error) {
	var allServices []*service.Service
	for _, s := range c.services {
		allServices = append(allServices, &service.Service{
			ID:   service.ServiceId(s.ID),
			Name: s.Name,
			RegisteringAgentAddress: s.Address,
			Tags: s.Tags,
		})
	}
	return allServices, nil
}

func (c ConsulStub) FailGetServicesForName(failOnName string) {
	c.failGetServicesForNames[failOnName] = true
}

func (c ConsulStub) FailRegisterForId(taskId apps.TaskId) {
	c.failRegisterForIds[taskId] = true
}

func (c ConsulStub) FailDeregisterByTaskForId(taskId apps.TaskId) {
	c.failDeregisterByTaskForIds[taskId] = true
}

func (c ConsulStub) FailDeregisterForId(serviceId service.ServiceId) {
	c.failDeregisterForIds[serviceId] = true
}

func (c ConsulStub) GetServices(name string) ([]*service.Service, error) {
	if _, ok := c.failGetServicesForNames[name]; ok {
		return nil, fmt.Errorf("Consul stub programmed to fail when getting services for name %s", name)
	}
	var services []*service.Service
	for _, s := range c.services {
		if s.Name == name && contains(s.Tags, c.consul.config.Tag) {
			services = append(services, &service.Service{
				ID:   service.ServiceId(s.ID),
				Name: s.Name,
				RegisteringAgentAddress: s.Address,
				Tags: s.Tags,
			})
		}
	}
	return services, nil
}

func (c *ConsulStub) Register(task *apps.Task, app *apps.App) error {
	if _, ok := c.failRegisterForIds[task.ID]; ok {
		return fmt.Errorf("Consul stub programmed to fail when registering task of id %s", task.ID.String())
	} else {
		serviceRegistration, err := c.consul.marathonTaskToConsulService(task, app)
		if err != nil {
			return err
		}
		c.services[service.ServiceId(serviceRegistration.ID)] = serviceRegistration
		return nil
	}
}

func (c *ConsulStub) ServiceName(app *apps.App) string {
	return c.consul.ServiceName(app)
}

func (c *ConsulStub) DeregisterByTask(taskId apps.TaskId, agent string) error {
	if _, ok := c.failDeregisterByTaskForIds[taskId]; ok {
		return fmt.Errorf("Consul stub programmed to fail when deregistering task of id %s", taskId.String())
	} else {
		for _, x := range c.servicesMatchingTask(c.services, taskId) {
			delete(c.services, service.ServiceId(x.ID))
		}
		return nil
	}
}

func (c *ConsulStub) Deregister(toDeregister *service.Service) error {
	if _, ok := c.failDeregisterForIds[toDeregister.ID]; ok {
		return fmt.Errorf("Consul stub programmed to fail when deregistering service of id %s", toDeregister.ID)
	}
	delete(c.services, toDeregister.ID)
	return nil
}

func (c *ConsulStub) servicesMatchingTask(services map[service.ServiceId]*consulapi.AgentServiceRegistration, taskId apps.TaskId) []*consulapi.AgentServiceRegistration {
	matching := []*consulapi.AgentServiceRegistration{}
	for _, s := range services {
		if s.ID == taskId.String() || contains(s.Tags, fmt.Sprintf("marathon-task:%s", taskId.String())) {
			matching = append(matching, s)
		}
	}
	return matching
}

func (c *ConsulStub) RegisteredTaskIds() []apps.TaskId {
	services, _ := c.GetAllServices()
	taskIds := []apps.TaskId{}
	for _, s := range services {
		taskId, _ := s.TaskId()
		taskIds = append(taskIds, taskId)
	}
	return taskIds
}
