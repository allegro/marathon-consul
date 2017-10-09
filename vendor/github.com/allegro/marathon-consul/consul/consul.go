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
	agents                  Agents
	config                  Config
	ignoredHealthCheckTypes []string
}

type ServicesProvider func(agent *consulapi.Client) ([]*service.Service, error)

func New(config Config) *Consul {
	return &Consul{
		agents:                  NewAgents(&config),
		config:                  config,
		ignoredHealthCheckTypes: ignoredHealthCheckTypesFromRawConfigEntry(config.IgnoredHealthChecks),
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
	return nil, errors.New("An error occurred getting services from Consul. Giving up")
}

func (c *Consul) getServicesUsingAgent(name string, agent *consulapi.Client) ([]*service.Service, error) {
	dcAwareQueries, err := dcAwareQueriesForAllDCs(agent)
	if err != nil {
		return nil, err
	}
	var allServices []*service.Service

	for _, dcAwareQuery := range dcAwareQueries {
		allConsulServices, _, err := agent.Catalog().Service(name, c.config.Tag, dcAwareQuery)
		if err != nil {
			return nil, err
		}
		allServices = append(allServices, consulServicesToServices(allConsulServices)...)
	}
	return allServices, nil
}

func dcAwareQueriesForAllDCs(agent *consulapi.Client) ([]*consulapi.QueryOptions, error) {
	datacenters, err := agent.Catalog().Datacenters()
	if err != nil {
		return nil, err
	}

	var queries []*consulapi.QueryOptions
	for _, dc := range datacenters {
		queries = append(queries, &consulapi.QueryOptions{
			Datacenter: dc,
		})
	}

	return queries, nil
}

func (c *Consul) GetAllServices() ([]*service.Service, error) {
	return c.getServicesUsingProviderWithRetriesOnAgentFailure(c.getAllServices)
}

func (c *Consul) getAllServices(agent *consulapi.Client) ([]*service.Service, error) {
	dcAwareQueries, err := dcAwareQueriesForAllDCs(agent)
	if err != nil {
		return nil, err
	}
	var allInstances []*service.Service

	for _, dcAwareQuery := range dcAwareQueries {
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
				allInstances = append(allInstances, consulServicesToServices(consulServiceInstances)...)
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
		EnableTagOverride:       consulService.ServiceEnableTagOverride,
	}
}

func consulServicesToServices(consulServices []*consulapi.CatalogService) []*service.Service {
	var allServices []*service.Service
	for _, c := range consulServices {
		allServices = append(allServices, consulServiceToService(c))
	}
	return allServices
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
	services, err := c.marathonTaskToConsulServices(task, app)
	if err != nil {
		return err
	}
	if value, ok := app.Labels[apps.MarathonConsulLabel]; ok && value == "true" {
		log.WithField("Id", app.ID).Warn("Warning! Application configuration is deprecated (labeled as `consul:true`). Support for special `true` value will be removed in the future!")
	}
	metrics.Time("consul.register", func() { err = c.registerMultipleServices(services) })
	if err != nil {
		metrics.Mark("consul.register.error")
	} else {
		metrics.Mark("consul.register.success")
	}
	return err
}

func (c *Consul) registerMultipleServices(services []*consulapi.AgentServiceRegistration) error {
	var registerErrors []error
	for _, s := range services {
		registerErr := c.register(s)
		if registerErr != nil {
			registerErrors = append(registerErrors, registerErr)
		}
	}

	return utils.MergeErrorsOrNil(registerErrors, "registering services")
}

func (c *Consul) register(service *consulapi.AgentServiceRegistration) error {
	agent, err := c.agents.GetAgent(service.Address)
	if err != nil {
		return err
	}
	fields := log.Fields{
		"Name":              service.Name,
		"Id":                service.ID,
		"Tags":              service.Tags,
		"Address":           service.Address,
		"Port":              service.Port,
		"EnableTagOverride": service.EnableTagOverride,
	}
	log.WithFields(fields).Info("Registering")

	err = agent.Agent().ServiceRegister(service)
	if err != nil {
		log.WithError(err).WithFields(fields).Error("Unable to register")
	}
	return err
}

func (c *Consul) DeregisterByTask(taskID apps.TaskID) error {
	services, err := c.findServicesByTaskID(taskID)
	if err != nil {
		return err
	} else if len(services) == 0 {
		log.WithField("Id", taskID).Warningf("Couldn't find any service matching task id")
		return nil
	}
	return c.deregisterMultipleServices(services, taskID)
}

func (c *Consul) deregisterMultipleServices(services []*service.Service, taskID apps.TaskID) error {
	var deregisterErrors []error
	for _, s := range services {
		deregisterErr := c.Deregister(s)
		if deregisterErr != nil {
			deregisterErrors = append(deregisterErrors, deregisterErr)
		}
	}

	return utils.MergeErrorsOrNil(deregisterErrors, fmt.Sprintf("deregistering by task %s", taskID))
}

func (c *Consul) findServicesByTaskID(searchedTaskID apps.TaskID) ([]*service.Service, error) {
	return c.getServicesUsingProviderWithRetriesOnAgentFailure(func(agent *consulapi.Client) ([]*service.Service, error) {
		dcAwareQueries, err := dcAwareQueriesForAllDCs(agent)
		if err != nil {
			return nil, err
		}

		var allFound []*service.Service
		searchedTag := service.MarathonTaskTag(searchedTaskID)
		for _, dcAwareQuery := range dcAwareQueries {
			consulServices, _, err := agent.Catalog().Services(dcAwareQuery)
			if err != nil {
				return nil, err
			}
			for consulService, tags := range consulServices {
				if contains(tags, searchedTag) {
					instancesForTask, _, err := agent.Catalog().Service(consulService, searchedTag, dcAwareQuery)
					if err != nil {
						return nil, err
					}
					allFound = append(allFound, consulServicesToServices(instancesForTask)...)
				}
			}
		}
		return allFound, nil
	})
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

func (c *Consul) marathonTaskToConsulServices(task *apps.Task, app *apps.App) ([]*consulapi.AgentServiceRegistration, error) {
	IP, err := utils.HostToIPv4(task.Host)
	if err != nil {
		return nil, err
	}
	serviceAddress := IP.String()
	checks := c.marathonToConsulChecks(task, app.HealthChecks, serviceAddress)

	var registrations []*consulapi.AgentServiceRegistration
	for _, intent := range app.RegistrationIntents(task, c.config.ConsulNameSeparator) {
		tags := append([]string{c.config.Tag}, intent.Tags...)
		tags = append(tags, service.MarathonTaskTag(task.ID))
		registrations = append(registrations, &consulapi.AgentServiceRegistration{
			ID:                c.serviceID(task, intent.Name, intent.Port),
			Name:              intent.Name,
			Port:              intent.Port,
			Address:           serviceAddress,
			Tags:              tags,
			Checks:            checks,
			EnableTagOverride: c.config.EnableTagOverride,
		})
	}
	return registrations, nil
}

func (c *Consul) serviceID(task *apps.Task, name string, port int) string {
	return fmt.Sprintf("%s_%s_%d", task.ID, name, port)
}

func (c *Consul) marathonToConsulChecks(task *apps.Task, healthChecks []apps.HealthCheck, serviceAddress string) consulapi.AgentServiceChecks {
	var checks = make(consulapi.AgentServiceChecks, 0, len(healthChecks))
	for _, check := range healthChecks {
		if contains(c.ignoredHealthCheckTypes, check.Protocol) {
			log.WithField("Id", task.AppID.String()).WithField("Address", serviceAddress).
				Info(fmt.Sprintf("Ignoring health check of type %s", check.Protocol))
			continue
		}

		port, err := getHealthCheckPort(check, *task)
		if err != nil {
			log.WithField("Id", task.AppID.String()).WithField("Address", serviceAddress).WithError(err).
				Warnf("Ignoring health check of type %s", check.Protocol)
			continue
		}

		if c := marathonToConsulCheck(task, check, serviceAddress, port); c != nil {
			checks = append(checks, c)
		}

	}
	return checks
}

func marathonToConsulCheck(task *apps.Task, check apps.HealthCheck, serviceAddress string, port int) *consulapi.AgentServiceCheck {
	consulCheck := &consulapi.AgentServiceCheck{
		Interval: fmt.Sprintf("%ds", check.IntervalSeconds),
		Timeout:  fmt.Sprintf("%ds", check.TimeoutSeconds),
		Status:   "passing",
	}

	switch check.Protocol {
	case "HTTP", "HTTPS", "MESOS_HTTP", "MESOS_HTTPS":
		path := "/"
		if check.Path != "" {
			path = check.Path
		}
		if parsedURL, err := url.ParseRequestURI(path); err == nil {
			if check.Protocol == "HTTP" || check.Protocol == "MESOS_HTTP" {
				parsedURL.Scheme = "http"
			} else {
				parsedURL.Scheme = "https"
			}
			parsedURL.Host = fmt.Sprintf("%s:%d", serviceAddress, port)
			consulCheck.HTTP = parsedURL.String()
			return consulCheck
		} else {
			log.WithError(err).
				WithField("Id", task.AppID.String()).
				WithField("Address", serviceAddress).
				Warnf("Could not parse provided path: %s", path)
		}
	case "TCP", "MESOS_TCP":
		consulCheck.TCP = fmt.Sprintf("%s:%d", serviceAddress, port)
		return consulCheck
	case "COMMAND":
		consulCheck.Script = substituteEnvironment(check.Command.Value, *task)
		return consulCheck
	default:
		log.WithField("Id", task.AppID.String()).WithField("Address", serviceAddress).
			Warnf("Unrecognized check protocol %s", check.Protocol)
	}
	return nil
}

func getHealthCheckPort(check apps.HealthCheck, task apps.Task) (int, error) {
	port := 0
	if check.Port != 0 {
		port = check.Port
	} else if check.PortIndex >= 0 && check.PortIndex < len(task.Ports) {
		port = task.Ports[check.PortIndex]
	} else {
		return 0, fmt.Errorf("Port index (%d) out of bounds should from range [0,%d)", check.PortIndex, len(task.Ports))
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("Port %d is invalid", port)
	}
	return port, nil
}

func substituteEnvironment(s string, task apps.Task) string {
	for i, p := range task.Ports {
		s = strings.Replace(s, fmt.Sprintf("$PORT%v", i), fmt.Sprintf("%v", p), -1)
	}
	s = strings.Replace(s, "$HOST", task.Host, -1)
	return s
}

func ignoredHealthCheckTypesFromRawConfigEntry(raw string) []string {
	ignoredTypes := make([]string, 0)
	for _, ignoredType := range strings.Split(strings.ToUpper(raw), ",") {
		var ignoredType = strings.TrimSpace(ignoredType)
		if ignoredType != "" {
			ignoredTypes = append(ignoredTypes, ignoredType)
		}
	}
	return ignoredTypes
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
