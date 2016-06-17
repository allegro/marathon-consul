package consul

import (
	"fmt"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
)

type ConsulServices interface {
	GetAllServices() ([]*consulapi.CatalogService, error)
	GetServices(name string) ([]*consulapi.CatalogService, error)
	Register(task *apps.Task, app *apps.App) error
	Deregister(serviceId apps.TaskId, agentAddress string) error
	GetAgent(agentAddress string) (*consulapi.Client, error)
}

type Consul struct {
	agents Agents
	config ConsulConfig
}

type ServicesProvider func(agent *consulapi.Client) ([]*consulapi.CatalogService, error)

func New(config ConsulConfig) *Consul {
	apps.CONSUL_NAME_SEPARATOR = config.ConsulNameSeparator
	return &Consul{
		agents: NewAgents(&config),
		config: config,
	}
}

func (c *Consul) GetServices(name string) ([]*consulapi.CatalogService, error) {
	return c.getServicesUsingProviderWithRetriesOnAgentFailure(func(agent *consulapi.Client) ([]*consulapi.CatalogService, error) {
		return c.getServicesUsingAgent(name, agent)
	})
}

func (c *Consul) getServicesUsingProviderWithRetriesOnAgentFailure(provide ServicesProvider) ([]*consulapi.CatalogService, error) {
	for retry := uint32(0); retry <= c.config.RequestRetries; retry++ {
		agent, err := c.agents.GetAnyAgent()
		if err != nil {
			return nil, err
		}
		if services, err := provide(agent.Client); err != nil {
			log.WithError(err).WithField("Address", agent.IP).
				Error("An error occurred getting services from Consul, retrying with another agent")
			if failures := agent.IncFailures(); failures > c.config.AgentFailuresTolerance {
				log.WithField("Address", agent.IP).WithField("Failures", failures).
					Warn("Removing agent due to too many failures")
				c.agents.RemoveAgent(agent.IP)
			}
		} else {
			agent.ClearFailures()
			return services, nil
		}
	}
	return nil, fmt.Errorf("An error occurred getting services from Consul. Giving up")
}

func (c *Consul) getServicesUsingAgent(name string, agent *consulapi.Client) ([]*consulapi.CatalogService, error) {
	datacenters, err := agent.Catalog().Datacenters()
	if err != nil {
		return nil, err
	}
	var allServices []*consulapi.CatalogService

	for _, dc := range datacenters {
		dcAwareQuery := &consulapi.QueryOptions{
			Datacenter: dc,
		}
		services, _, err := agent.Catalog().Service(name, c.config.Tag, dcAwareQuery)
		if err != nil {
			return nil, err
		}
		allServices = append(allServices, services...)
	}
	return allServices, nil
}

func (c *Consul) GetAllServices() ([]*consulapi.CatalogService, error) {
	return c.getServicesUsingProviderWithRetriesOnAgentFailure(c.getAllServices)
}

func (c *Consul) getAllServices(agent *consulapi.Client) ([]*consulapi.CatalogService, error) {
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
			if contains(tags, c.config.Tag) {
				serviceInstances, _, err := agent.Catalog().Service(service, c.config.Tag, dcAwareQuery)
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

func (c *Consul) Register(task *apps.Task, app *apps.App) error {
	service, err := c.marathonTaskToConsulService(task, app)
	if err != nil {
		return err
	}
	if value, ok := app.Labels[apps.MARATHON_CONSUL_LABEL]; ok && value == "true" {
		log.WithField("Id", app.ID).Warn("Warning! Application configuration is deprecated (labeled as `consul:true`). Support for special `true` value will be removed in the future!")
	}
	metrics.Time("consul.register", func() { err = c.register(service) })
	if err != nil {
		metrics.Mark("consul.register.error")
	} else {
		metrics.Mark("consul.register.success")
	}
	return err
}

func (c *Consul) register(service *consulapi.AgentServiceRegistration) error {
	agent, err := c.agents.GetAgent(service.Address)
	if err != nil {
		return err
	}
	fields := log.Fields{
		"Name":    service.Name,
		"Id":      service.ID,
		"Tags":    service.Tags,
		"Address": service.Address,
		"Port":    service.Port,
	}
	log.WithFields(fields).Info("Registering")

	err = agent.Agent().ServiceRegister(service)
	if err != nil {
		log.WithError(err).WithFields(fields).Error("Unable to register")
	}
	return err
}

func (c *Consul) Deregister(serviceId apps.TaskId, agentAddress string) error {
	var err error
	metrics.Time("consul.deregister", func() { err = c.deregister(serviceId, agentAddress) })
	if err != nil {
		metrics.Mark("consul.deregister.error")
	} else {
		metrics.Mark("consul.deregister.success")
	}
	return err
}

func (c *Consul) deregister(serviceId apps.TaskId, agentAddress string) error {
	agent, err := c.agents.GetAgent(agentAddress)
	if err != nil {
		return err
	}

	log.WithField("Id", serviceId).WithField("Address", agentAddress).Info("Deregistering")

	err = agent.Agent().ServiceDeregister(serviceId.String())
	if err != nil {
		log.WithError(err).WithField("Id", serviceId).WithField("Address", agentAddress).Error("Unable to deregister")
	}
	return err
}

func (c *Consul) GetAgent(agentAddress string) (*consulapi.Client, error) {
	return c.agents.GetAgent(agentAddress)
}

func (c *Consul) marathonTaskToConsulService(task *apps.Task, app *apps.App) (*consulapi.AgentServiceRegistration, error) {
	IP, err := utils.HostToIPv4(task.Host)
	if err != nil {
		return nil, err
	}
	serviceAddress := IP.String()

	return &consulapi.AgentServiceRegistration{
		ID:      task.ID.String(),
		Name:    app.ConsulServiceName(),
		Port:    task.Ports[0],
		Address: serviceAddress,
		Tags:    c.marathonLabelsToConsulTags(app.Labels),
		Checks:  c.marathonToConsulChecks(task, app.HealthChecks, serviceAddress),
	}, nil
}

func (c *Consul) marathonToConsulChecks(task *apps.Task, healthChecks []apps.HealthCheck, serviceAddress string) consulapi.AgentServiceChecks {
	var checks consulapi.AgentServiceChecks = make(consulapi.AgentServiceChecks, 0, len(healthChecks))

	for _, check := range healthChecks {
		switch check.Protocol {
		case "HTTP", "HTTPS":
			if parsedUrl, err := url.ParseRequestURI(check.Path); err == nil {
				parsedUrl.Scheme = strings.ToLower(check.Protocol)
				parsedUrl.Host = fmt.Sprintf("%s:%d", serviceAddress, task.Ports[check.PortIndex])

				checks = append(checks, &consulapi.AgentServiceCheck{
					HTTP:     parsedUrl.String(),
					Interval: fmt.Sprintf("%ds", check.IntervalSeconds),
					Timeout:  fmt.Sprintf("%ds", check.TimeoutSeconds),
				})
			} else {
				log.WithError(err).
					WithField("Id", task.AppID.String()).
					WithField("Address", serviceAddress).
					Warn(fmt.Sprintf("Could not parse provided path: %s", check.Path))
			}
		case "TCP":
			checks = append(checks, &consulapi.AgentServiceCheck{
				TCP:      fmt.Sprintf("%s:%d", serviceAddress, task.Ports[check.PortIndex]),
				Interval: fmt.Sprintf("%ds", check.IntervalSeconds),
				Timeout:  fmt.Sprintf("%ds", check.TimeoutSeconds),
			})
		case "COMMAND":
			checks = append(checks, &consulapi.AgentServiceCheck{
				Script:   check.Command.Value,
				Interval: fmt.Sprintf("%ds", check.IntervalSeconds),
				Timeout:  fmt.Sprintf("%ds", check.TimeoutSeconds),
			})
		default:
			log.WithField("Id", task.AppID.String()).WithField("Address", serviceAddress).
				Warn(fmt.Sprintf("Unrecognized check protocol %s", check.Protocol))
		}
	}
	return checks
}

func (c *Consul) marathonLabelsToConsulTags(labels map[string]string) []string {
	tags := []string{c.config.Tag}
	for key, value := range labels {
		if value == "tag" {
			tags = append(tags, key)
		}
	}
	return tags
}
