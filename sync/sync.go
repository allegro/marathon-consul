package sync

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/service"
)

type Sync struct {
	config              Config
	marathon            marathon.Marathoner
	serviceRegistry     service.ServiceRegistry
	syncStartedListener SyncStartedListener
}

type SyncStartedListener func(apps []*apps.App)

var noopSyncStartedListener = func(apps []*apps.App) {}

func New(config Config, marathon marathon.Marathoner, serviceRegistry service.ServiceRegistry, syncStartedListener SyncStartedListener) *Sync {
	return &Sync{config, marathon, serviceRegistry, syncStartedListener}
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

	s.syncStartedListener(apps)

	services, err := s.serviceRegistry.GetAllServices()
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

func (s *Sync) deregisterConsulServicesNotFoundInMarathon(marathonApps []*apps.App, services []*service.Service) {
	runningTasks := s.marathonTaskIdsSet(marathonApps)
	for _, service := range services {
		taskIdInTag, err := service.TaskId()
		taskIdNotFoundInTag := err != nil
		if taskIdNotFoundInTag {
			log.WithField("Id", service.ID).WithError(err).
				Warn("Couldn't extract marathon task id, deregistering to have sync reregister it properly")
		}

		if _, isRunning := runningTasks[taskIdInTag]; !isRunning || taskIdNotFoundInTag {
			err := s.serviceRegistry.Deregister(service)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"Id":      service.ID,
					"Address": service.RegisteringAgentAddress,
				}).Error("Can't deregister service")
			}
		} else {
			log.WithField("Id", service.ID).Debug("Service is running")
		}
	}
}

func (s *Sync) registerAppTasksNotFoundInConsul(marathonApps []*apps.App, services []*service.Service) {
	registrationsUnderTaskIds := s.taskIdsInConsulServices(services)
	for _, app := range marathonApps {
		if !app.IsConsulApp() {
			log.WithField("Id", app.ID).Debug("Not a Consul app, skipping registration")
			continue
		}
		expectedRegistrations := app.RegistrationIntentsNumber()
		for _, task := range app.Tasks {
			if registrations := registrationsUnderTaskIds[task.ID]; registrations != expectedRegistrations {
				if registrations != 0 {
					log.WithField("Id", task.ID).WithField("HasRegistrations", registrations).
					WithField("ExpectedRegistrations", expectedRegistrations).Info("Registering missing service registrations")
				}
				if task.IsHealthy() {
					err := s.serviceRegistry.Register(&task, app)
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

func (s *Sync) taskIdsInConsulServices(services []*service.Service) map[apps.TaskId]int {
	serviceCounters := make(map[apps.TaskId]int)
	for _, service := range services {
		if taskId, err := service.TaskId(); err == nil {
			serviceCounters[taskId] += 1
		}
	}
	return serviceCounters
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
