package main

import (
	"github.com/CiscoCloud/marathon-forwarder/config"
	"log"
	"net/http"
	"runtime"
)

func main() {
	config := config.New()
	apiConfig, err := config.Registry.Config()
	if err != nil {
		log.Fatal(err.Error())
	}

	forwarder, err := NewForwarder(apiConfig, runtime.NumCPU())
	if err != nil {
		log.Fatal(err.Error())
	}
	forwarder.Verbose = config.Verbose

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{*forwarder}
	http.HandleFunc("/events", forwarderHandler.Handle)

	log.Printf(`listening on "%s"`, config.Web.Listen)
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
