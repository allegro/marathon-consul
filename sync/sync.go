package sync

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	service "github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	consul "github.com/hashicorp/consul/api"
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

	apps, err := s.marathon.ConsulApps()
	if err != nil {
		return fmt.Errorf("Can't get Marathon apps: %v", err)
	}

	s.addAgentNodes(apps)

	services, err := s.service.GetAllServices()
	if err != nil {
		return fmt.Errorf("Can't get Consul services: %v", err)
	}

	s.deregisterConsulServicesNotFoundInMarathon(apps, services)
	s.registerAppTasksNotFoundInConsul(apps, services)

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
		log.WithField("Leader", leader).WithField("Node", s.config.Leader).Debug("Node is not a leader, skipping sync")
		return false, nil
	}
	log.WithField("Node", s.config.Leader).Debug("Node has leadership")
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

func (s *Sync) addAgentNodes(apps []*apps.App) {
	nodes := make(map[string]struct{})
	var exists struct{}
	for _, app := range apps {
		if !app.IsConsulApp() {
			continue
		}
		for _, task := range app.Tasks {
			nodes[task.Host] = exists
		}
	}
	for node := range nodes {
		_, err := s.service.GetAgent(node)
		if err != nil {
			log.WithError(err).WithField("Node", node).Error("Can't add agent node")
		}
	}
}

func (s *Sync) deregisterConsulServicesNotFoundInMarathon(marathonApps []*apps.App, consulServices []*consul.CatalogService) {
	marathonTaskIdsSet := s.marathonTaskIdsSet(marathonApps)
	for _, service := range consulServices {
		serviceId := apps.TaskId(service.ServiceID)
		if _, ok := marathonTaskIdsSet[serviceId]; !ok {
			err := s.service.Deregister(serviceId, service.Address)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"Id":      service.ServiceID,
					"Address": service.Address,
				}).Error("Can't deregister service")
			}
		} else {
			log.WithField("Id", service.ServiceID).Debug("Service is running")
		}
	}
}

func (s *Sync) registerAppTasksNotFoundInConsul(marathonApps []*apps.App, consulServices []*consul.CatalogService) {
	consulServicesIdsSet := s.consulServiceIdsSet(consulServices)
	for _, app := range marathonApps {
		if !app.IsConsulApp() {
			log.WithField("Id", app.ID).Debug("Not a Consul app, skipping registration")
			continue
		}
		for _, task := range app.Tasks {
			if _, ok := consulServicesIdsSet[task.ID]; !ok {
				if task.IsHealthy() {
					err := s.service.Register(&task, app)
					if err != nil {
						log.WithError(err).WithField("Id", task.ID).Error("Can't register task")
					}
				} else {
					log.WithField("Id", task.ID).Debug("Task is not healthy. Not Registering")
				}
			} else {
				log.WithField("Id", task.ID).Debug("Task already registered in Consul")
			}
		}
	}
}

func (s *Sync) consulServiceIdsSet(services []*consul.CatalogService) map[apps.TaskId]struct{} {
	servicesSet := make(map[apps.TaskId]struct{})
	var exists struct{}
	for _, service := range services {
		servicesSet[apps.TaskId(service.ServiceID)] = exists
	}
	return servicesSet
}

func (s *Sync) marathonTaskIdsSet(marathonApps []*apps.App) map[apps.TaskId]struct{} {
	tasksSet := make(map[apps.TaskId]struct{})
	var exists struct{}
	for _, app := range marathonApps {
		for _, task := range app.Tasks {
			tasksSet[task.ID] = exists
		}
	}
	return tasksSet
}
