package consul

import (
	"errors"
	"fmt"
	"strings"

	"github.com/allegro/marathon-consul/apps"
	consulapi "github.com/hashicorp/consul/api"
)

type ConsulStub struct {
	services                   map[string]*consulapi.AgentServiceRegistration
	failGetServicesForNames    map[string]bool
	failRegisterForIds         map[apps.TaskId]bool
	failDeregisterByTaskForIds map[apps.TaskId]bool
	failDeregisterForIds       map[string]bool
	consul                     *Consul
}

func NewConsulStub() *ConsulStub {
	return NewConsulStubWithTag("marathon")
}

func NewConsulStubWithTag(tag string) *ConsulStub {
	return &ConsulStub{
		services:                   make(map[string]*consulapi.AgentServiceRegistration),
		failGetServicesForNames:    make(map[string]bool),
		failRegisterForIds:         make(map[apps.TaskId]bool),
		failDeregisterByTaskForIds: make(map[apps.TaskId]bool),
		failDeregisterForIds:       make(map[string]bool),
		consul:                     New(ConsulConfig{Tag: tag, ConsulNameSeparator: "."}),
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

func (c ConsulStub) FailGetServicesForName(failOnName string) {
	c.failGetServicesForNames[failOnName] = true
}

func (c ConsulStub) FailRegisterForId(taskId apps.TaskId) {
	c.failRegisterForIds[taskId] = true
}

func (c ConsulStub) FailDeregisterByTaskForId(taskId apps.TaskId) {
	c.failDeregisterByTaskForIds[taskId] = true
}

func (c ConsulStub) FailDeregisterForId(serviceId string) {
	c.failDeregisterForIds[serviceId] = true
}

func (c ConsulStub) GetServices(name string) ([]*consulapi.CatalogService, error) {
	if _, ok := c.failGetServicesForNames[name]; ok {
		return nil, fmt.Errorf("Consul stub programmed to fail when getting services for name %s", name)
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

func (c *ConsulStub) Register(task *apps.Task, app *apps.App) error {
	if _, ok := c.failRegisterForIds[task.ID]; ok {
		return fmt.Errorf("Consul stub programmed to fail when registering task of id %s", task.ID.String())
	} else {
		service, err := c.consul.marathonTaskToConsulService(task, app)
		if err != nil {
			return err
		}
		c.services[service.ID] = service
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
			delete(c.services, x.ID)
		}
		return nil
	}
}

func (c *ConsulStub) Deregister(serviceId string, agent string) error {
	if _, ok := c.failDeregisterForIds[serviceId]; ok {
		return fmt.Errorf("Consul stub programmed to fail when deregistering service of id %s", serviceId)
	}
	delete(c.services, serviceId)
	return nil
}

func (s *ConsulStub) ServiceTaskId(service *consulapi.CatalogService) (apps.TaskId, error) {
	for _, tag := range service.ServiceTags {
		if strings.HasPrefix(tag, "marathon-task:") {
			return apps.TaskId(strings.TrimPrefix(tag, "marathon-task:")), nil
		}
	}
	return apps.TaskId(""), errors.New("marathon-task tag missing")
}

func (c *ConsulStub) servicesMatchingTask(services map[string]*consulapi.AgentServiceRegistration, taskId apps.TaskId) []*consulapi.AgentServiceRegistration {
	matching := []*consulapi.AgentServiceRegistration{}
	for _, s := range services {
		if (s.ID == taskId.String() || contains(s.Tags, fmt.Sprintf("marathon-task:%s", taskId.String()))) {
			matching = append(matching, s)
		}
	}
	return matching
}

func (c *ConsulStub) RegisteredTaskIds() []apps.TaskId {
	services, _ := c.GetAllServices()
	taskIds := []apps.TaskId{}
	for _, consulService := range services {
		taskId, _ := c.ServiceTaskId(consulService)
		taskIds = append(taskIds, taskId)
	}
	return taskIds
}

func (c *ConsulStub) GetAgent(agentAddress string) (*consulapi.Client, error) {
	return nil, nil
}
