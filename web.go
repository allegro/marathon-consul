package main

import (
	"bytes"
	"errors"
	"fmt"
	service "github.com/CiscoCloud/marathon-consul/consul-services"
	"github.com/CiscoCloud/marathon-consul/events"
	marathon "github.com/CiscoCloud/marathon-consul/marathon"
	"github.com/CiscoCloud/marathon-consul/tasks"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

type ForwardHandler struct {
	service  service.Consul
	marathon marathon.Marathoner
}

func (fh *ForwardHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(500)
		fmt.Fprintln(w, "could not read request body")
		return
	}

	eventType, err := events.EventType(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		return
	}

	switch eventType {
	case "app_terminated_event":
		log.WithField("eventType", "app_terminated_event").Info("handling event")
		fh.HandleTerminationEvent(w, body)
	case "status_update_event":
		log.WithField("eventType", "status_update_event").Info("handling event")
		fh.HandleStatusEvent(w, body)
	case "health_status_changed_event":
		log.WithField("eventType", "health_status_changed_event").Info("handling event")
		fh.HandleHealthStatusEvent(w, body)
	default:
		log.WithField("eventType", eventType).Info("not handling event")
		w.WriteHeader(200)
		fmt.Fprintf(w, "cannot handle %s\n", eventType)
	}
	log.Debug(string(body))
}

func (fh *ForwardHandler) HandleTerminationEvent(w http.ResponseWriter, body []byte) {
	event, err := events.ParseEvent(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.Printf("[ERROR] body generated error: %s", err.Error())
		return
	}

	// app_terminated_event only has one app in it, so we will just take care of
	// it instead of looping
	app := event.Apps()[0]

	tasks, err := fh.marathon.Tasks(app.ID)
	for _, task := range tasks {
		fh.service.Deregister(task.ID, task.Host)
	}

	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.WithError(err).WithField("Name", app.ID).Error("There where problems processing request")
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
}

func (fh *ForwardHandler) HandleHealthStatusEvent(w http.ResponseWriter, body []byte) {
	body = bytes.Replace(body, []byte("taskId"), []byte("id"), -1)
	taskHealthChange, err := tasks.ParseTaskHealthChange(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.WithError(err).Error("[ERROR] body generated error: %s")
		return
	}

	if taskHealthChange.Alive {
		appId := taskHealthChange.AppID
		app, err := fh.marathon.App(appId)
		tasks := app.Tasks

		if err != nil {
			log.WithField("ID", taskHealthChange.ID).WithError(err).Error("There was a problem obtaining app info")
			return
		}

		if value, ok := app.Labels["consul"]; !ok || value != "true" {
			log.WithField("Name", appId).Info("App should not be registered in Consul")
			return
		}

		healthCheck := app.HealthChecks
		labels := app.Labels

		for _, task := range tasks {
			if task.ID == taskHealthChange.ID {
				if service.IsTaskHealthy(task.HealthCheckResults) {
					fh.service.Register(service.MarathonTaskToConsulService(task, healthCheck, labels))
				}
			}
			break
		}
	}
}

func (fh *ForwardHandler) HandleStatusEvent(w http.ResponseWriter, body []byte) {
	// for every other use of Tasks, Marathon uses the "id" field for the task ID.
	// Here, it uses "taskId", with most of the other fields being equal. We'll
	// just swap "taskId" for "id" in the body so that we can successfully parse
	// incoming events.
	body = bytes.Replace(body, []byte("taskId"), []byte("id"), -1)

	task, err := tasks.ParseTask(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.WithError(err).WithField("Body", body).Error("[ERROR] body generated error")
		return
	}

	switch task.TaskStatus {
	case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
		fh.service.Deregister(task.ID, task.Host)
	case "TASK_STAGING", "TASK_STARTING", "TASK_RUNNING":
	default:
		err = errors.New("unknown task status")
	}

	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.WithError(err).WithField("ID", task.ID).Error("There where problems processing request")
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
}
