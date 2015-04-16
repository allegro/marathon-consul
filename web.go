package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

type ForwardHandler struct {
	Forwarder
}

func (fh *ForwardHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(500)
		fmt.Fprintln(w, "could not read request body")
		return
	}

	apps, err := ParseApps(body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err.Error())
		return
	}

	errs := fh.ForwardApps(apps)
	resp := ""
	respCode := 200
	for _, err := range errs {
		if err != nil {
			respCode = 500
			resp = fmt.Sprintf("%s%s\n", resp, err.Error())
		}
	}
	if resp == "" {
		resp = "OK\n"
	}

	w.WriteHeader(respCode)
	fmt.Fprint(w, resp)
}
