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
	syncStartedListener startedListener
}

type startedListener func(apps []*apps.App)

func New(config Config, marathon marathon.Marathoner, serviceRegistry service.ServiceRegistry, syncStartedListener startedListener) *Sync {
	return &Sync{config, marathon, serviceRegistry, syncStartedListener}
}

func (s *Sync) StartSyncServicesJob() {
	if !s.config.Enabled {
		log.Info("Marathon-consul sync disabled")
		return
	}

	log.WithFields(log.Fields{
		"Interval": s.config.Interval,
		"Leader":   s.config.Leader,
		"Force":    s.config.Force,
	}).Info("Marathon-consul sync job started")

	ticker := time.NewTicker(s.config.Interval.Duration)
	go func() {
		if err := s.SyncServices(); err != nil {
			log.WithError(err).Error("An error occured while performing sync")
		}
		for range ticker.C {
			if err := s.SyncServices(); err != nil {
				log.WithError(err).Error("An error occured while performing sync")
			}
		}
	}()
	return
}

func (s *Sync) SyncServices() error {
	var err error
	metrics.Time("sync.services", func() { err = s.syncServices() })
	return err
}

func (s *Sync) syncServices() error {
	if check, err := s.shouldPerformSync(); !check {
		metrics.Clear()
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

	registerCount, registerErrorsCount := s.registerAppTasksNotFoundInConsul(apps, services)
	deregisterCount, deregisterErrorsCount := s.deregisterConsulServicesNotFoundInMarathon(apps, services)

	metrics.UpdateGauge("sync.register.success", int64(registerCount))
	metrics.UpdateGauge("sync.register.error", int64(registerErrorsCount))
	metrics.UpdateGauge("sync.deregister.success", int64(deregisterCount))
	metrics.UpdateGauge("sync.deregister.error", int64(deregisterErrorsCount))

	log.Infof("Syncing services finished. Stats, registerd: %d (failed: %d), deregister: %d (failed: %d).",
		registerCount, registerErrorsCount, deregisterCount, deregisterErrorsCount)
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

func (s *Sync) deregisterConsulServicesNotFoundInMarathon(marathonApps []*apps.App, services []*service.Service) (deregisterCount int, errorCount int) {
	runningTasks := marathonTaskIdsSet(marathonApps)
	for _, service := range services {
		logFields := log.Fields{
			"Id":      service.ID,
			"Address": service.RegisteringAgentAddress,
			"Sync":    true,
		}
		if taskIDInTag, err := service.TaskId(); err != nil {
			log.WithField("Id", service.ID).WithError(err).
				Warn("Couldn't extract marathon task id, deregistering since sync should have reregistered it already")
			if err := s.serviceRegistry.Deregister(service); err != nil {
				log.WithError(err).WithFields(logFields).Error("Can't deregister service")
				errorCount++
			} else {
				deregisterCount++
			}
		} else if _, isRunning := runningTasks[taskIDInTag]; !isRunning {
			// Check latest marathon state to prevent deregistration of live service.
			tasks, err := s.marathon.Tasks(taskIDInTag.AppID())
			if err != nil {
				log.WithError(err).WithFields(logFields).
					Error("Can't get fresh info about app tasks. Will deregister this service.")
			}

			_, taskIsRunning := apps.FindTaskByID(taskIDInTag, tasks)

			if !taskIsRunning {
				if err := s.serviceRegistry.Deregister(service); err != nil {
					log.WithError(err).WithFields(logFields).Error("Can't deregister service")
					errorCount++
				} else {
					deregisterCount++
				}
			}
		} else {
			log.WithField("Id", service.ID).Debug("Service is running")
		}
	}
	return
}

func (s *Sync) registerAppTasksNotFoundInConsul(marathonApps []*apps.App, services []*service.Service) (registerCount int, errorCount int) {
	registrationsUnderTaskIds := taskIdsInConsulServices(services)
	for _, app := range marathonApps {
		if !app.IsConsulApp() {
			log.WithField("Id", app.ID).Debug("Not a Consul app, skipping registration")
			continue
		}
		expectedRegistrations := app.RegistrationIntentsNumber()
		for _, task := range app.Tasks {
			registrations := registrationsUnderTaskIds[task.ID]
			logFields := log.Fields{
				"Id":                    task.ID,
				"HasRegistrations":      registrations,
				"ExpectedRegistrations": expectedRegistrations,
				"Sync":                  true,
			}
			if registrations < expectedRegistrations {
				if registrations != 0 {
					log.WithFields(logFields).Info("Registering missing service registrations")
				}
				if task.IsHealthy() {
					err := s.serviceRegistry.Register(&task, app)
					if err != nil {
						log.WithError(err).WithFields(logFields).Error("Can't register task")
						errorCount++
					} else {
						registerCount++
					}
				} else {
					log.WithFields(logFields).Debug("Task is not healthy. Not Registering")
				}
			} else if registrations > expectedRegistrations {
				log.WithFields(logFields).Warn("Skipping task with excess registrations")
			} else {
				log.WithFields(logFields).Debug("Task already registered in Consul")
			}
		}
	}
	return
}

func taskIdsInConsulServices(services []*service.Service) map[apps.TaskID]int {
	serviceCounters := make(map[apps.TaskID]int)
	for _, service := range services {
		if taskID, err := service.TaskId(); err == nil {
			serviceCounters[taskID]++
		}
	}
	return serviceCounters
}

func marathonTaskIdsSet(marathonApps []*apps.App) map[apps.TaskID]struct{} {
	tasksSet := make(map[apps.TaskID]struct{})
	var exists struct{}
	for _, app := range marathonApps {
		for _, task := range app.Tasks {
			tasksSet[task.ID] = exists
		}
	}
	return tasksSet
}