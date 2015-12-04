package sync

import (
	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	service "github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	consul "github.com/hashicorp/consul/api"
	"time"
)

type Sync struct {
	marathon marathon.Marathoner
	service  service.ConsulServices
}

func New(marathon marathon.Marathoner, service service.ConsulServices) *Sync {
	return &Sync{marathon, service}
}

func (s *Sync) StartSyncServicesJob(interval time.Duration) *time.Ticker {
	log.WithField("Interval", interval).Info("Marathon-consul sync job started")
	ticker := time.NewTicker(interval)
	go func() {
		s.SyncServices()
		for {
			select {
			case <-ticker.C:
				s.SyncServices()
			}
		}
	}()
	return ticker
}

func (s *Sync) SyncServices() error {
	var err error
	metrics.Time("sync.services", func() { err = s.syncServices() })
	return err
}

func (s *Sync) syncServices() error {
	log.Info("Syncing services")

	apps, err := s.marathon.Apps()
	if err != nil {
		return err
	}

	s.registerMarathonApps(apps)

	services, err := s.service.GetAllServices()
	if err != nil {
		log.WithError(err).Error("Can't get all Consul services")
		return err
	}

	s.deregisterConsulServicesThatAreNotInMarathonApps(apps, services)

	log.Info("Syncing services finished")
	return nil
}

func (s *Sync) registerMarathonApps(apps []*apps.App) {

	for _, app := range apps {
		tasks := app.Tasks
		healthCheck := app.HealthChecks
		labels := app.Labels

		if value, ok := app.Labels["consul"]; !ok || value != "true" {
			log.WithField("APP", app.ID).Debug("App should not be registered in Consul")
			continue
		}

		for _, task := range tasks {
			if service.IsTaskHealthy(task.HealthCheckResults) {
				service := service.MarathonTaskToConsulService(task, healthCheck, labels)
				err := s.service.Register(service)
				if err != nil {
					log.WithError(err).WithField("ID", task.ID).Error("Can't register task")
				}
			} else {
				log.WithFields(log.Fields{
					"APP": app.ID, "ID": task.ID,
				}).Debug("Task is not healthy. Not Registering")
			}
		}
	}
}

func (s Sync) deregisterConsulServicesThatAreNotInMarathonApps(apps []*apps.App, services []*consul.CatalogService) {
	//	TODO: Change it to map implementation
	for _, instance := range services {
		found := false
		for _, app := range apps {
			for _, task := range app.Tasks {
				found = found || instance.ServiceID == task.ID
			}
		}
		if !found {
			err := s.service.Deregister(instance.ServiceID, instance.Node)
			if err != nil {
				log.WithError(err).WithField("ID", instance.ServiceID).Error("Can't deregister service")
			}
		}
	}
}
