package marathon

import (
	"github.com/CiscoCloud/marathon-consul/consul"
	service "github.com/CiscoCloud/marathon-consul/consul-services"
	log "github.com/Sirupsen/logrus"
)

type MarathonSync struct {
	marathon Marathoner
	consul   consul.Consul
	service  service.Consul
}

func NewMarathonSync(marathon Marathoner, consul consul.Consul, service service.Consul) *MarathonSync {
	return &MarathonSync{marathon, consul, service}
}

func (m *MarathonSync) SyncServices() error {
	//	TODO: Add metrics about registered and unregistered services
	log.Info("syncing services")
	apps, err := m.marathon.Apps()
	if err != nil {
		return err
	}

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

	services, err := m.service.GetAllServices()
	if err != nil {
		log.WithError(err).Error("Cant get all Consul services")
		return err
	}

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
	log.Info("syncing services finished")
	return nil
}

func (m *MarathonSync) Sync() error {

	// apps
	log.Info("syncing apps")
	apps, err := m.marathon.Apps()
	if err != nil {
		return err
	}
	err = m.consul.SyncApps(apps)
	if err != nil {
		return err
	}

	// tasks
	log.Info("syncing tasks")
	for _, app := range apps {
		log.WithField("app", app.ID).Debug("syncing tasks for app")
		tasks, err := m.marathon.Tasks(app.ID)
		if err != nil {
			return err
		}
		err = m.consul.SyncTasks(app.ID, tasks)
		if err != nil {
			return err
		}
	}

	return nil
}
