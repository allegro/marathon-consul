package sync

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	service "github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/tasks"
	consul "github.com/hashicorp/consul/api"
	"os"
	"time"
)

type Sync struct {
	config   Config
	marathon marathon.Marathoner
	service  service.ConsulServices
}

func New(config Config, marathon marathon.Marathoner, service service.ConsulServices) *Sync {
	return &Sync{config, marathon, service}
}

func (s *Sync) StartSyncServicesJob() *time.Ticker {
	if !s.config.Enabled {
		log.Info("Marathon-consul sync disabled")
		return nil
	}

	log.WithFields(log.Fields{
		"Interval": s.config.Interval,
		"Leader":   s.config.Leader,
		"Force":    s.config.Force,
	}).Info("Marathon-consul sync job started")

	ticker := time.NewTicker(s.config.Interval)
	go func() {
		s.SyncServices()
		for {
			select {
			case <-ticker.C:
				err := s.SyncServices()
				if err != nil {
					log.WithError(err).Error("An error occured while performing sync")
				}
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
	if check, err := s.shouldPerformSync(); !check {
		return err
	}
	log.Info("Syncing services started")

	apps, err := s.marathon.Apps()
	if err != nil {
		return fmt.Errorf("Can't get Marathon apps: %v", err)
	}
	s.registerMarathonApps(apps)

	services, err := s.service.GetAllServices()
	// TODO use services info to create new consul agents list, make (de)registration call only when needed
	if err != nil {
		return fmt.Errorf("Can't get all Consul services: %v", err)
	}

	s.deregisterConsulServicesThatAreNotInMarathonApps(apps, services)

	log.Info("Syncing services finished")
	return nil
}

func (s *Sync) shouldPerformSync() (bool, error) {
	if s.config.Force {
		log.Debug("Forcing sync")
		return true, nil
	}
	leader, err := s.marathon.Leader()
	if err != nil {
		return false, fmt.Errorf("Could not get Marathon leader: %v", err)
	}
	if s.config.Leader == "" {
		if err = s.resolveHostname(); err != nil {
			return false, fmt.Errorf("Could not resolve hostname: %v", err)
		}
	}
	if leader != s.config.Leader {
		log.WithField("Leader", leader).WithField("Node", s.config.Leader).Info("Node is not a leader, skipping sync")
		return false, nil
	}
	log.WithField("Node", s.config.Leader).Info("Node has leadership")
	return true, nil
}

func (s *Sync) resolveHostname() error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	s.config.Leader = fmt.Sprintf("%s:8080", hostname)
	log.WithField("Leader", s.config.Leader).Info("Marathon-consul sync leader mode set to resolved hostname")
	return nil
}

func (s *Sync) registerMarathonApps(apps []*apps.App) {

	for _, app := range apps {
		tasks := app.Tasks
		healthCheck := app.HealthChecks
		labels := app.Labels

		if !app.IsConsulApp() {
			log.WithField("Id", app.ID).Debug("App should not be registered in Consul")
			continue
		}

		for _, task := range tasks {
			if service.IsTaskHealthy(task.HealthCheckResults) {
				service := service.MarathonTaskToConsulService(task, healthCheck, labels)
				err := s.service.Register(service)
				if err != nil {
					log.WithError(err).WithField("Id", task.ID).Error("Can't register task")
				}
			} else {
				log.WithField("Id", task.ID).Debug("Task is not healthy. Not Registering")
			}
		}
	}
}

func (s Sync) deregisterConsulServicesThatAreNotInMarathonApps(apps []*apps.App, services []*consul.CatalogService) {
	marathonTasksIdSet := make(map[tasks.Id]struct{})
	var exist struct{}
	for _, app := range apps {
		for _, task := range app.Tasks {
			marathonTasksIdSet[task.ID] = exist
		}
	}
	for _, instance := range services {
		instanceId := tasks.Id(instance.ServiceID)
		if _, ok := marathonTasksIdSet[instanceId]; !ok {
			err := s.service.Deregister(instanceId, instance.Address)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"Id":      instanceId,
					"Address": instance.Address,
				}).Error("Can't deregister service")
			}
		}
	}
}
