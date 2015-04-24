package main

import (
	"encoding/json"
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
	Forwarder
	Verbose bool
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

	if !(event.Type == "api_post_event" || event.Type == "deployment_info") {
		w.WriteHeader(200)
		fmt.Fprintln(w, "this endpoint only accepts api_post_event and deployment_info")
		if fh.Verbose {
			log.Printf("received '%s' event, not handling", event.Type)
		}
		return
	}

	if fh.Verbose {
		log.Printf("received '%s' event, handling", event.Type)
	}

	apps, err := ParseApps(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		if fh.Verbose {
			log.Printf("body generated error: %s", err.Error())
			log.Println(string(body))
		}
		return
	}

	errs := fh.ForwardApps(apps)
	resp := ""
	respCode := 200
	for _, err := range errs {
		if err != nil {
			respCode = 500
			resp = fmt.Sprintf("%s%s\n", resp, err.Error())
			if fh.Verbose {
				log.Printf("response generated error: %s", err.Error())
			}
		}
	}
	if resp == "" {
		resp = "OK\n"
	}

	w.WriteHeader(respCode)
	fmt.Fprint(w, resp)
}
