package main

import (
	"github.com/CiscoCloud/marathon-consul/config"
	"log"
	"net/http"
)

func main() {
	config := config.New()
	apiConfig, err := config.Registry.Config()
	if err != nil {
		log.Fatal(err.Error())
	}

	kv, err := NewKV(apiConfig)
	if err != nil {
		log.Fatal(err.Error())
	}
	kv.Prefix = config.Registry.Prefix

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{*kv}
	http.HandleFunc("/events", forwarderHandler.Handle)

	log.Printf(`listening on "%s"`, config.Web.Listen)
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
