package consul

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/service"
	"github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
)

type Consul struct {
	agents Agents
	config ConsulConfig
}

type ServicesProvider func(agent *consulapi.Client) ([]*service.Service, error)

func New(config ConsulConfig) *Consul {
	return &Consul{
		agents: NewAgents(&config),
		config: config,
	}
}

func (c *Consul) GetServices(name string) ([]*service.Service, error) {
	return c.getServicesUsingProviderWithRetriesOnAgentFailure(func(agent *consulapi.Client) ([]*service.Service, error) {
		return c.getServicesUsingAgent(name, agent)
	})
}

func (c *Consul) getServicesUsingProviderWithRetriesOnAgentFailure(provide ServicesProvider) ([]*service.Service, error) {
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

func (c *Consul) getServicesUsingAgent(name string, agent *consulapi.Client) ([]*service.Service, error) {
	datacenters, err := agent.Catalog().Datacenters()
	if err != nil {
		return nil, err
	}
	var allServices []*service.Service

	for _, dc := range datacenters {
		dcAwareQuery := &consulapi.QueryOptions{
			Datacenter: dc,
		}
		allConsulServices, _, err := agent.Catalog().Service(name, c.config.Tag, dcAwareQuery)
		if err != nil {
			return nil, err
		}
		for _, consulService := range allConsulServices {
			allServices = append(allServices, consulServiceToService(consulService))
		}
	}
	return allServices, nil
}

func (c *Consul) GetAllServices() ([]*service.Service, error) {
	return c.getServicesUsingProviderWithRetriesOnAgentFailure(c.getAllServices)
}

func (c *Consul) getAllServices(agent *consulapi.Client) ([]*service.Service, error) {
	datacenters, err := agent.Catalog().Datacenters()
	if err != nil {
		return nil, err
	}
	var allInstances []*service.Service

	for _, dc := range datacenters {
		dcAwareQuery := &consulapi.QueryOptions{
			Datacenter: dc,
		}
		consulServices, _, err := agent.Catalog().Services(dcAwareQuery)
		if err != nil {
			return nil, err
		}
		for consulService, tags := range consulServices {
			if contains(tags, c.config.Tag) {
				consulServiceInstances, _, err := agent.Catalog().Service(consulService, c.config.Tag, dcAwareQuery)
				if err != nil {
					return nil, err
				}
				for _, consulServiceInstance := range consulServiceInstances {
					allInstances = append(allInstances, consulServiceToService(consulServiceInstance))
				}
			}
		}
	}
	return allInstances, nil
}

func consulServiceToService(consulService *consulapi.CatalogService) *service.Service {
	return &service.Service{
		ID:   service.ServiceId(consulService.ServiceID),
		Name: consulService.ServiceName,
		Tags: consulService.ServiceTags,
		RegisteringAgentAddress: consulService.Address,
	}
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

func (c *Consul) DeregisterByTask(taskId apps.TaskId, agentAddress string) error {
	service, err := c.findServiceByTaskId(taskId)
	if err != nil {
		return err
	}
	return c.Deregister(service)
}

func (c *Consul) findServiceByTaskId(searchedTaskId apps.TaskId) (*service.Service, error) {
	services, err := c.GetAllServices()
	if err != nil {
		return nil, err
	}
	for _, s := range services {
		taskId, err := s.TaskId()
		if err == nil && taskId == searchedTaskId {
			return s, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Couldn't find service matching task id %s", searchedTaskId))
}

func (c *Consul) Deregister(toDeregister *service.Service) error {
	var err error
	metrics.Time("consul.deregister", func() { err = c.deregister(toDeregister) })
	if err != nil {
		metrics.Mark("consul.deregister.error")
	} else {
		metrics.Mark("consul.deregister.success")
	}
	return err
}

func (c *Consul) deregister(toDeregister *service.Service) error {
	agent, err := c.agents.GetAgent(toDeregister.RegisteringAgentAddress)
	if err != nil {
		return err
	}

	log.WithField("Id", toDeregister.ID).WithField("Address", toDeregister.RegisteringAgentAddress).Info("Deregistering")

	err = agent.Agent().ServiceDeregister(toDeregister.ID.String())
	if err != nil {
		log.WithError(err).WithField("Id", toDeregister.ID).WithField("Address", toDeregister.RegisteringAgentAddress).Error("Unable to deregister")
	}
	return err
}

func (c *Consul) ServiceName(app *apps.App) string {
	appConsulName := app.ConsulName()
	serviceName := c.marathonAppNameToConsulServiceName(appConsulName)
	if serviceName == "" {
		log.WithField("AppId", app.ID.String()).WithField("ConsulServiceName", appConsulName).
			Warn("Warning! Invalid Consul service name provided for app. Will use default app name instead.")
		return c.marathonAppNameToConsulServiceName(app.ID.String())
	}
	return serviceName
}

func (c *Consul) marathonAppNameToConsulServiceName(name string) string {
	return strings.Replace(strings.Trim(strings.TrimSpace(name), "/"), "/", c.config.ConsulNameSeparator, -1)
}

func (c *Consul) marathonTaskToConsulService(task *apps.Task, app *apps.App) (*consulapi.AgentServiceRegistration, error) {
	IP, err := utils.HostToIPv4(task.Host)
	if err != nil {
		return nil, err
	}
	serviceAddress := IP.String()

	name := c.ServiceName(app)
	port := task.Ports[0]
	serviceID := c.serviceId(task, name, port)
	tags := c.marathonLabelsToConsulTags(app.Labels)
	tags = append(tags, service.MarathonTaskTag(task.ID))

	return &consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    name,
		Port:    port,
		Address: serviceAddress,
		Tags:    tags,
		Checks:  c.marathonToConsulChecks(task, app.HealthChecks, serviceAddress),
	}, nil
}

func (c *Consul) serviceId(task *apps.Task, name string, port int) string {
	return fmt.Sprintf("%s_%s_%d", task.ID, name, port)
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

func (c *Consul) AddAgentsFromApps(apps []*apps.App) {
	for _, app := range apps {
		if !app.IsConsulApp() {
			continue
		}
		for _, task := range app.Tasks {
			err := c.AddAgent(task.Host)
			if err != nil {
				log.WithError(err).WithField("Node", task.Host).Error("Can't add agent node")
			}
		}
	}
}

func (c *Consul) AddAgent(agentAddress string) error {
	_, err := c.agents.GetAgent(agentAddress)
	return err
}
