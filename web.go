package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

type Event struct {
	Type string `json:"eventType"`
}

type ForwardHandler struct {
	kv PutDeleter
}

func (fh *ForwardHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(500)
		fmt.Fprintln(w, "could not read request body")
		return
	}

	event := Event{}
	err = json.Unmarshal(body, &event)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		return
	}

	switch event.Type {
	case "api_post_event", "deployment_info":
		fh.HandleAppEvent(w, body)
	case "status_update_event":
		fh.HandleStatusEvent(w, body)
	default:
		w.WriteHeader(200)
		fmt.Fprintf(w, "cannot handle %s\n", event.Type)
	}
}

func (fh *ForwardHandler) HandleAppEvent(w http.ResponseWriter, body []byte) {
	apps, err := ParseApps(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		log.Printf("[ERROR] body generated error: %s", err.Error())
		return
	}

	resp := ""
	respCode := 200
	for _, app := range apps {
		if app.Active {
			_, err = fh.kv.Put(app.KV())
		} else {
			_, err = fh.kv.Delete(app.Key())
		}
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

func (fh *ForwardHandler) HandleStatusEvent(w http.ResponseWriter, body []byte) {
	task, err := ParseTask(body)
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
