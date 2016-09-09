package web

import (
	"bytes"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/service"
	"github.com/allegro/marathon-consul/utils"
)

type event struct {
	timestamp time.Time
	eventType string
	body      []byte
}

type eventHandler struct {
	id              int
	serviceRegistry service.ServiceRegistry
	marathon        marathon.Marathoner
	eventQueue      <-chan event
}

type stopEvent struct{}

func newEventHandler(id int, serviceRegistry service.ServiceRegistry, marathon marathon.Marathoner, eventQueue <-chan event) *eventHandler {
	return &eventHandler{
		id:              id,
		serviceRegistry: serviceRegistry,
		marathon:        marathon,
		eventQueue:      eventQueue,
	}
}

func (fh *eventHandler) Start() chan<- stopEvent {
	var e event
	process := func() {
		err := fh.handleEvent(e.eventType, e.body)
		if err != nil {
			metrics.Mark("events.processing.error")
		} else {
			metrics.Mark("events.processing.succes")
		}
	}

	quitChan := make(chan stopEvent)
	log.WithField("Id", fh.id).Println("Starting worker")
	go func() {
		for {
			select {
			case e = <-fh.eventQueue:
				metrics.Mark(fmt.Sprintf("events.handler.%d", fh.id))
				metrics.UpdateGauge("events.queue.len", int64(len(fh.eventQueue)))
				metrics.UpdateGauge("events.queue.delay_ns", time.Since(e.timestamp).Nanoseconds())
				metrics.Time("events.processing."+e.eventType, process)
			case <-quitChan:
				log.WithField("Id", fh.id).Info("Stopping worker")
			}
		}
	}()
	return quitChan
}

func (fh *eventHandler) handleEvent(eventType string, body []byte) error {

	body = replaceTaskIDWithID(body)

	switch eventType {
	case "status_update_event":
		return fh.handleStatusEvent(body)
	case "health_status_changed_event":
		return fh.handleHealthStatusEvent(body)
	case "unhealthy_task_kill_event":
		return fh.handleUnhealthyTaskKillEvent(body)
	case "deployment_info":
		return fh.handleDeploymentInfo(body)
	case "deployment_step_success":
		return fh.handleDeploymentStepSuccess(body)
	default:
		log.WithField("EventType", eventType).Debug("Not handled event type")
		return nil
	}
}

func (fh *eventHandler) handleHealthStatusEvent(body []byte) error {
	taskHealthChange, err := events.ParseTaskHealthChange(body)
	if err != nil {
		log.WithError(err).Error("Body generated error")
		return err
	}

	log.WithFields(
		log.Fields{
			"Id":         taskHealthChange.ID,
			"TaskStatus": taskHealthChange.TaskStatus,
		}).Info("Got HealthStatusEvent")

	if !taskHealthChange.Alive {
		log.WithField("Id", taskHealthChange.ID).Debug("Task is not alive. Not registering")
		return nil
	}

	appID := taskHealthChange.AppID
	app, err := fh.marathon.App(appID)
	if err != nil {
		log.WithField("Id", taskHealthChange.ID).WithError(err).Error("There was a problem obtaining app info")
		return err
	}

	if !app.IsConsulApp() {
		err = fmt.Errorf("%s is not consul app. Missing consul label", app.ID)
		log.WithField("Id", taskHealthChange.ID).WithError(err).Debug("Skipping app registration in Consul")
		return nil
	}

	tasks := app.Tasks

	task, err := findTaskByID(taskHealthChange.ID, tasks)
	if err != nil {
		log.WithField("Id", taskHealthChange.ID).WithError(err).Error("Task not found")
		return err
	}

	if task.IsHealthy() {
		err := fh.serviceRegistry.Register(&task, app)
		if err != nil {
			log.WithField("Id", task.ID).WithError(err).Error("There was a problem registering task")
			return err
		}
		return nil
	}
	log.WithField("Id", task.ID).Debug("Task is not healthy. Not registering")
	return nil
}

func (fh *eventHandler) handleStatusEvent(body []byte) error {
	task, err := apps.ParseTask(body)

	if err != nil {
		log.WithError(err).WithField("Body", body).Error("Could not parse event body")
		return err
	}

	log.WithFields(log.Fields{
		"Id":         task.ID,
		"TaskStatus": task.TaskStatus,
	}).Info("Got StatusEvent")

	switch task.TaskStatus {
	case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
		return fh.deregister(task.ID)
	default:
		log.WithFields(log.Fields{
			"Id":         task.ID,
			"taskStatus": task.TaskStatus,
		}).Debug("Not handled task status")
		return nil
	}
}

func (fh *eventHandler) handleUnhealthyTaskKillEvent(body []byte) error {
	task, err := apps.ParseTask(body)

	if err != nil {
		log.WithError(err).WithField("Body", body).Error("Could not parse event body")
		return err
	}

	log.WithFields(log.Fields{
		"Id": task.ID,
	}).Info("Got Unhealthy TaskKilled Event")

	return fh.deregister(task.ID)
}

//This handler is used when an application is stopped
func (fh *eventHandler) handleDeploymentInfo(body []byte) error {
	deploymentEvent, err := events.ParseDeploymentEvent(body)

	if err != nil {
		log.WithError(err).WithField("Body", body).Error("Could not parse event body")
		return err
	}

	errors := []error{}
	for _, app := range deploymentEvent.StoppedConsulApps() {
		errors = append(errors, fh.deregisterAllAppServices(app)...)
	}
	return utils.MergeErrorsOrNil(errors, "handling deregistration")
}

//This handler is used when an application is restarted and renamed
func (fh *eventHandler) handleDeploymentStepSuccess(body []byte) error {
	deploymentEvent, err := events.ParseDeploymentEvent(body)

	if err != nil {
		log.WithError(err).WithField("Body", body).Error("Could not parse event body")
		return err
	}

	errors := []error{}
	for _, app := range deploymentEvent.RenamedConsulApps() {
		errors = append(errors, fh.deregisterAllAppServices(app)...)
	}
	return utils.MergeErrorsOrNil(errors, "handling deregistration")
}

func (fh *eventHandler) deregisterAllAppServices(app *apps.App) []error {
	errors := []error{}
	serviceNames := fh.serviceRegistry.ServiceNames(app)

	log.WithField("AppId", app.ID).WithField("ServiceNames", serviceNames).Info("Deregistering all services")

	var allServices []*service.Service
	for _, name := range serviceNames {
		services, err := fh.serviceRegistry.GetServices(name)
		if err != nil {
			log.WithField("Id", app.ID).WithError(err).Error("There was a problem getting Consul services")
			errors = append(errors, err)
			return errors
		}
		allServices = append(allServices, services...)
	}

	if len(allServices) == 0 {
		log.WithField("AppId", app.ID).WithField("ServiceNames", serviceNames).Info("No matching Consul services found")
		return errors
	}

	for _, service := range allServices {
		taskID, err := service.TaskId()
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if taskID.AppID() == app.ID {
			err = fh.deregister(taskID)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	return errors
}

func (fh *eventHandler) deregister(taskID apps.TaskID) error {
	err := fh.serviceRegistry.DeregisterByTask(taskID)
	if err != nil {
		log.WithField("Id", taskID).WithError(err).Error("There was a problem deregistering task")
	}
	return err
}

func findTaskByID(id apps.TaskID, tasks []apps.Task) (apps.Task, error) {
	for _, task := range tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return apps.Task{}, fmt.Errorf("Task %s not found", id)
}

// for every other use of Tasks, Marathon uses the "id" field for the task ID.
// Here, it uses "taskId", with most of the other fields being equal. We'll
// just swap "taskId" for "id" in the body so that we can successfully parse
// incoming events.
func replaceTaskIDWithID(body []byte) []byte {
	return bytes.Replace(body, []byte("taskId"), []byte("id"), -1)
}
