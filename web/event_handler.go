package web

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	service "github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/events"
	marathon "github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/tasks"
	"io/ioutil"
	"net/http"
)

type EventHandler struct {
	service  service.ConsulServices
	marathon marathon.Marathoner
}

func NewEventHandler(service service.ConsulServices, marathon marathon.Marathoner) *EventHandler {
	return &EventHandler{
		service:  service,
		marathon: marathon,
	}
}

func (fh *EventHandler) Handle(w http.ResponseWriter, r *http.Request) {
	metrics.Time("events.response", func() { fh.handle(w, r) })
}

func (fh *EventHandler) handle(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Debug("Malformed request")
		fh.handleBadRequest(err, w)
		return
	}
	log.WithField("Body", string(body)).Debug("Received")

	eventType, err := events.EventType(body)
	if err != nil {
		fh.handleBadRequest(err, w)
		return
	}

	fh.markEventRequest(eventType)

	log.WithField("EventType", eventType).Debug("Received event")

	switch eventType {
	case "app_terminated_event":
		fh.handleTerminationEvent(w, body)
	case "status_update_event":
		fh.handleStatusEvent(w, body)
	case "health_status_changed_event":
		fh.handleHealthStatusEvent(w, body)
	default:
		fh.handleBadRequest(fmt.Errorf("Cannot handle %s", eventType), w)
	}

	fh.markSuccess()
}

func (fh *EventHandler) handleTerminationEvent(w http.ResponseWriter, body []byte) {
	event, err := events.ParseEvent(body)
	if err != nil {
		fh.handleBadRequest(err, w)
		return
	}

	// app_terminated_event only has one app in it, so we will just take care of
	// it instead of looping
	app := event.Apps()[0]
	log.WithField("Id", app.ID).Info("Got TerminationEvent")

	tasks, err := fh.marathon.Tasks(app.ID)
	if err != nil {
		log.WithField("Id", app.ID).WithError(err).Error("There was a problem obtaining tasks for app")
		fh.handleError(err, w)
		return
	}

	errors := []error{}
	for _, task := range tasks {
		err = fh.service.Deregister(task.ID, task.Host)
		if err != nil {
			log.WithField("Id", task.ID).WithError(err).Error("There was a problem deregistering task")
			errors = append(errors, err)
		}
	}
	if len(errors) != 0 {
		errMessage := fmt.Sprintf("%d errors occured deregistering %d services:", len(errors), len(tasks))
		for i, err := range errors {
			errMessage = fmt.Sprintf("%s\n%d: %s", errMessage, i, err.Error())
		}
		err = fmt.Errorf(errMessage)
		fh.handleError(err, w)
		log.WithError(err).WithField("Id", app.ID).Error("There where problems processing request")
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
}

func (fh *EventHandler) handleHealthStatusEvent(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	taskHealthChange, err := events.ParseTaskHealthChange(body)
	if err != nil {
		log.WithError(err).Error("Body generated error")
		fh.handleBadRequest(err, w)
		return
	}

	log.WithFields(
		log.Fields{
			"Id":         taskHealthChange.ID,
			"TaskStatus": taskHealthChange.TaskStatus,
		}).Info("Got HealthStatusEvent")

	if !taskHealthChange.Alive {
		err := fmt.Errorf("Task %s is not healthy. Not registering", taskHealthChange.ID)
		log.WithField("Id", taskHealthChange.ID).WithError(err).Debug("Task is not healthy. Not registering")
		fh.handleBadRequest(err, w)
		return
	}

	appId := taskHealthChange.AppID
	app, err := fh.marathon.App(appId)
	if err != nil {
		log.WithField("Id", taskHealthChange.ID).WithError(err).Error("There was a problem obtaining app info")
		fh.handleError(err, w)
		return
	}
	tasks := app.Tasks

	if value, ok := app.Labels["consul"]; !ok || value != "true" {
		err = fmt.Errorf("%s is not consul app. Missing consul:true label", app.ID)
		log.WithField("Id", taskHealthChange.ID).WithError(err).Debug("App should not be registered in Consul")
		fh.handleBadRequest(err, w)
		return
	}

	healthCheck := app.HealthChecks
	labels := app.Labels

	task, err := findTaskById(taskHealthChange.ID, tasks)
	if err != nil {
		log.WithField("Id", taskHealthChange.ID).WithError(err).Error("Task not found")
		fh.handleError(err, w)
		return
	}

	if service.IsTaskHealthy(task.HealthCheckResults) {
		err := fh.service.Register(service.MarathonTaskToConsulService(task, healthCheck, labels))
		if err != nil {
			log.WithField("Id", task.ID).WithError(err).Error("There was a problem registering task")
			fh.handleError(err, w)
		} else {
			w.WriteHeader(200)
			fmt.Fprintln(w, "OK")
		}
	} else {
		err := fmt.Errorf("Task %s is not healthy. Not registering", task.ID)
		log.WithField("Id", task.ID).WithError(err).Debug("Task is not healthy. Not registering")
		fh.handleBadRequest(err, w)
	}
}

func findTaskById(id string, tasks_ []tasks.Task) (tasks.Task, error) {
	for _, task := range tasks_ {
		if task.ID == id {
			return task, nil
		}
	}
	return tasks.Task{}, fmt.Errorf("Task %s not found", id)
}

func (fh *EventHandler) handleStatusEvent(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	task, err := tasks.ParseTask(body)
	if err != nil {
		log.WithError(err).WithField("Body", body).Error("[ERROR] body generated error")
		fh.handleBadRequest(err, w)
	} else {
		log.WithFields(log.Fields{
			"Id":         task.ID,
			"TaskStatus": task.TaskStatus,
		}).Info("Got StatusEvent")
		switch task.TaskStatus {
		case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
			fh.service.Deregister(task.ID, task.Host)
			w.WriteHeader(200)
			fmt.Fprintln(w, "OK")
		default:
			log.WithFields(log.Fields{
				"taskStatus": task.TaskStatus,
				"Id":         task.ID,
			}).Info("not handling event")
			fh.handleBadRequest(fmt.Errorf("Not Handling task %s with status %s", task.ID, task.TaskStatus), w)
		}
	}
}

func replaceTaskIdWithId(body []byte) []byte {
	// for every other use of Tasks, Marathon uses the "id" field for the task ID.
	// Here, it uses "taskId", with most of the other fields being equal. We'll
	// just swap "taskId" for "id" in the body so that we can successfully parse
	// incoming events.
	return bytes.Replace(body, []byte("taskId"), []byte("id"), -1)
}

func (fh *EventHandler) markEventRequest(event string) {
	metrics.Mark("events.requests." + event)
}

func (fh *EventHandler) markSuccess() {
	metrics.Mark("events.response.success")
}

func (fh *EventHandler) handleError(err error, w http.ResponseWriter) {
	metrics.Mark("events.response.error.500")
	w.WriteHeader(500)
	log.WithError(err).Debug("Returning 500 due to error")
	fmt.Fprintln(w, err.Error())
}

func (fh *EventHandler) handleBadRequest(err error, w http.ResponseWriter) {
	metrics.Mark("events.response.error.400")
	w.WriteHeader(400)
	log.WithError(err).Debug("Returning 400 due to malformed request")
	fmt.Fprintln(w, err.Error())
}
