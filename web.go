package main

import (
	"bytes"
	"errors"
	"strings"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/consul"
	marathon "github.com/CiscoCloud/marathon-consul/marathon"
	service "github.com/CiscoCloud/marathon-consul/consul-services"
	"github.com/CiscoCloud/marathon-consul/events"
	"github.com/CiscoCloud/marathon-consul/tasks"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"

	"net/http"
	"github.com/CiscoCloud/mesos-consul/registry"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

type ForwardHandler struct {
	consul   consul.Consul
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
	case "api_post_event", "deployment_info":
		log.WithField("eventType", eventType).Info("handling event")
		fh.HandleAppEvent(w, body)
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

func (fh *ForwardHandler) HandleAppEvent(w http.ResponseWriter, body []byte) {
	event, err := events.ParseEvent(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.Printf("[ERROR] body generated error: %s", err.Error())
		return
	}

	resp := ""
	respCode := 200
	for _, app := range event.Apps() {
		err = fh.consul.UpdateApp(app)
		if err != nil {
			resp += err.Error() + "\n"
			log.Printf("[ERROR] response generated error: %s", err.Error())
			respCode = 500
		}
	}

	if resp == "" {
		resp = "OK\n"
	}

	w.WriteHeader(respCode)
	fmt.Fprint(w, resp)
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
	err = fh.consul.DeleteApp(event.Apps()[0])
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.Printf("[ERROR] response generated error: %s", err.Error())
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
}

func (fh *ForwardHandler) HandleHealthStatusEvent(w http.ResponseWriter, body []byte) {
	//	TODO: Implement me (Register task on "alive":true)
	body = bytes.Replace(body, []byte("taskId"), []byte("id"), -1)
	taskHealthChange, err := tasks.ParseTaskHealthChange(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.Printf("[ERROR] body generated error: %s", err.Error())
		return
	}


	if (taskHealthChange.Alive) {
		tasks, _ := fh.marathon.Tasks(taskHealthChange.AppID)
		for _, task := range tasks {
			if task.ID == taskHealthChange.ID {
				service := &registry.Service{
					ID: task.ID,
					Name: appIdToServiceName(task.AppID),
					Port: task.Ports[0], /*By default app should use ist 1st port*/
					Address: task.Host,
//					TODO: Pass labels as tags
					Tags: []string{},
//					TODO: Pass marathon checks as checks
					Check: &registry.Check{},
					Agent: task.Host,
				}
				fh.service.Register(service)
			}
		}

	}
}

func appIdToServiceName(appId string) (serviceId string) {
	serviceId = strings.Replace(strings.Trim(appId, "/"), "/", ".", -1)
	return serviceId
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
		log.Printf("[ERROR] body generated error: %s", err.Error())
		return
	}

	switch task.TaskStatus {
	case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
		fh.service.Deregister(task.ID, task.Host)
		err = fh.consul.DeleteTask(task)
	case "TASK_STAGING", "TASK_STARTING", "TASK_RUNNING":
		//		TODO: Noting (Task will be registered when it's healthy)
		err = fh.consul.UpdateTask(task)
	default:
		err = errors.New("unknown task status")
	}

	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
}
