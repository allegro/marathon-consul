package main

import (
	"errors"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/events"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"io/ioutil"
	"log"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

type ForwardHandler struct {
	kv      PutDeleter
	Verbose bool
	Debug   bool
}

func (fh *ForwardHandler) LogVerbose(s string) {
	if fh.Verbose {
		log.Printf("[INFO] %s\n", s)
	}
}

func (fh *ForwardHandler) LogDebug(s string) {
	if fh.Debug {
		log.Printf("[DEBUG] %s\n", s)
	}
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
		fh.LogVerbose(fmt.Sprintf("handling \"%s\"", eventType))
		fh.HandleAppEvent(w, body)
	case "app_terminated_event":
		fh.LogVerbose("handling \"app_terminated_event\"")
		fh.HandleTerminationEvent(w, body)
	case "status_update_event":
		fh.LogVerbose("handling \"sdeployment_infotatus_update_event\"")
		fh.HandleStatusEvent(w, body)
	default:
		fh.LogVerbose(fmt.Sprintf("not handling \"%s\"", eventType))
		w.WriteHeader(200)
		fmt.Fprintf(w, "cannot handle %s\n", eventType)
	}
	fh.LogDebug(string(body))
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
		_, err = fh.kv.Put(app.KV())
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
	_, err = fh.kv.Delete(event.Apps()[0].Key())
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.Printf("[ERROR] response generated error: %s", err.Error())
	} else {
		w.WriteHeader(200)
		fmt.Fprintln(w, "OK")
	}
}

func (fh *ForwardHandler) HandleStatusEvent(w http.ResponseWriter, body []byte) {
	task, err := tasks.ParseTask(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.Printf("[ERROR] body generated error: %s", err.Error())
		return
	}

	switch task.TaskStatus {
	case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
		_, err = fh.kv.Delete(task.Key())
	case "TASK_STAGING", "TASK_STARTING", "TASK_RUNNING":
		_, err = fh.kv.Put(task.KV())
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
