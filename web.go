package main

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

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

type ForwardHandler struct {
	service  service.ConsulServices
	marathon marathon.Marathoner
}

func (fh *ForwardHandler) Handle(w http.ResponseWriter, r *http.Request) {

	fh.markRequest()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Debug("Malformed request")
		fh.handleBadRequest(err, w)
		return
	}
	log.Debug(string(body))

	eventType, err := events.EventType(body)
	if err != nil {
		fh.handleBadRequest(err, w)
		return
	}

	fh.markEventRequest(eventType)

	switch eventType {
	case "app_terminated_event":
		log.WithField("eventType", "app_terminated_event").Info("handling event")
		fh.handleTerminationEvent(w, body)
	case "status_update_event":
		log.WithField("eventType", "status_update_event").Info("handling event")
		fh.handleStatusEvent(w, body)
	case "health_status_changed_event":
		log.WithField("eventType", "health_status_changed_event").Info("handling event")
		fh.handleHealthStatusEvent(w, body)
	default:
		log.WithField("eventType", eventType).Debug("not handling event")
		fh.handleBadRequest(fmt.Errorf("cannot handle %s", eventType), w)
	}

	fh.markResponse()
}

func (fh *ForwardHandler) handleTerminationEvent(w http.ResponseWriter, body []byte) {
	event, err := events.ParseEvent(body)
	if err != nil {
		fh.handleBadRequest(err, w)
		return
	}

	// app_terminated_event only has one app in it, so we will just take care of
	// it instead of looping
	app := event.Apps()[0]

	tasks, err := fh.marathon.Tasks(app.ID)
	if err != nil {
		log.WithField("APP", app.ID).WithError(err).Error("There was a problem obtaining tasks for app")
		fh.handleError(err, w)
		return
	}

	errors := []error{}
	for _, task := range tasks {
		err = fh.service.Deregister(task.ID, task.Host)
		if err != nil {
			log.WithField("ID", task.ID).WithError(err).Error("There was a problem deregistering task")
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
		log.WithError(err).WithField("APP", app.ID).Error("There where problems processing request")
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
}

func (fh *ForwardHandler) handleHealthStatusEvent(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	taskHealthChange, err := events.ParseTaskHealthChange(body)
	if err != nil {
		log.WithError(err).Error("Body generated error")
		fh.handleBadRequest(err, w)
		return
	}

	if !taskHealthChange.Alive {
		err := fmt.Errorf("Task %s is not healthy. Not registering", taskHealthChange.ID)
		log.WithField("ID", taskHealthChange.ID).WithError(err).Debug("Task is not healthy. Not registering")
		fh.handleBadRequest(err, w)
		return
	}

	appId := taskHealthChange.AppID
	app, err := fh.marathon.App(appId)
	if err != nil {
		log.WithField("ID", taskHealthChange.ID).WithError(err).Error("There was a problem obtaining app info")
		fh.handleError(err, w)
		return
	}
	tasks := app.Tasks

	if value, ok := app.Labels["consul"]; !ok || value != "true" {
		log.WithFields(log.Fields{
			"APP": appId,
			"ID":  taskHealthChange.ID,
		}).Debug("App should not be registered in Consul")
		fh.handleBadRequest(fmt.Errorf("%s is not consul app. Missing consul:true label", app.ID), w)
		return
	}

	healthCheck := app.HealthChecks
	labels := app.Labels

	task, err := findTaskById(taskHealthChange.ID, tasks)
	if err != nil {
		log.WithField("ID", taskHealthChange.ID).WithError(err).Error("Task not found")
		fh.handleError(err, w)
		return
	}

	if service.IsTaskHealthy(task.HealthCheckResults) {
		err := fh.service.Register(service.MarathonTaskToConsulService(task, healthCheck, labels))
		if err != nil {
			log.WithField("ID", task.ID).WithError(err).Error("There was a problem registering task")
			fh.handleError(err, w)
		} else {
			w.WriteHeader(200)
			fmt.Fprintln(w, "OK")
		}
	} else {
		err := fmt.Errorf("Task %s is not healthy. Not registering", task.ID)
		log.WithField("ID", task.ID).WithError(err).Debug("Task is not healthy. Not registering")
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

func (fh *ForwardHandler) handleStatusEvent(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	task, err := tasks.ParseTask(body)
	if err != nil {
		log.WithError(err).WithField("Body", body).Error("[ERROR] body generated error")
		fh.handleBadRequest(err, w)
	} else {
		switch task.TaskStatus {
		case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
			fh.service.Deregister(task.ID, task.Host)
			w.WriteHeader(200)
			fmt.Fprintln(w, "OK")
		default:
			log.WithFields(log.Fields{
				"taskStatus": task.TaskStatus,
				"ID":         task.ID,
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

func (fh *ForwardHandler) markRequest() {
	metrics.Mark("events.requests")
}

func (fh *ForwardHandler) markEventRequest(event string) {
	metrics.Mark("events.requests." + event)
}

func (fh *ForwardHandler) markResponse() {
	metrics.Mark("events.response")
}

func (fh *ForwardHandler) handleError(err error, w http.ResponseWriter) {
	metrics.Mark("events.error")
	w.WriteHeader(500)
	fmt.Fprintln(w, err.Error())
}

func (fh *ForwardHandler) handleBadRequest(err error, w http.ResponseWriter) {
	metrics.Mark("events.bad_request")
	w.WriteHeader(400)
	fmt.Fprintln(w, err.Error())
}
