package marathon

import (
	"github.com/CiscoCloud/marathon-consul/apps"
	service "github.com/CiscoCloud/marathon-consul/consul"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/CiscoCloud/marathon-consul/metrics"
)

type MarathonSync struct {
	marathon Marathoner
	service  service.ConsulServices
}

func NewMarathonSync(marathon Marathoner, service service.ConsulServices) *MarathonSync {
	return &MarathonSync{marathon, service}
}

func (m *MarathonSync) SyncServices() error {
	var err error
	metrics.Time("sync.services", func() { err = m.syncServices() })
	return err
}

func (m *MarathonSync) syncServices() error {
	//	TODO: Add metrics about registered and unregistered services
	log.Info("syncing services")

	apps, err := m.marathon.Apps()
	if err != nil {
		return err
	}

	m.registerMarathonApps(apps)

	services, err := m.service.GetAllServices()
	if err != nil {
		log.WithError(err).Error("Cant get all Consul services")
		return err
	}

	m.deregisterConsulServicesThatAreNotInMarathonApps(apps, services)

	log.Info("syncing services finished")
	return nil
}

func (m *MarathonSync) registerMarathonApps(apps []*apps.App) {

	for _, app := range apps {
		tasks := app.Tasks
		healthCheck := app.HealthChecks
		labels := app.Labels

		if value, ok := app.Labels["consul"]; !ok || value != "true" {
			log.WithField("Name", app.ID).Info("App should not be registered in Consul")
			continue
		}

		for _, task := range tasks {
			if service.IsTaskHealthy(task.HealthCheckResults) {
				m.service.Register(service.MarathonTaskToConsulService(task, healthCheck, labels))
			} else {
				log.WithFields(log.Fields{
					"Name": app.ID, "ID": task.ID,
				}).Info("Task is not healthy. Not Registering")
			}
		}
	}
}

func (m MarathonSync) deregisterConsulServicesThatAreNotInMarathonApps(apps []*apps.App, services []*consul.CatalogService) {
	//	TODO: Change it to map implementation
	for _, instance := range services {
		found := false
		for _, app := range apps {
			for _, task := range app.Tasks {
				found = found || instance.ServiceID == task.ID
			}
		}
		if !found {
			m.service.Deregister(instance.ServiceID, instance.Node)
		}
	}
}
