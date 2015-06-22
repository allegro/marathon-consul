package main

import (
	"github.com/CiscoCloud/marathon-consul/config"
	"github.com/CiscoCloud/marathon-consul/consul"
	"log"
	"net/http"
)

const Name = "marathon-consul"
const Version = "0.1.0"

func main() {
	config := config.New()
	apiConfig, err := config.Registry.Config()
	if err != nil {
		log.Fatal(err.Error())
	}

	kv, err := consul.NewKV(apiConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	consul := consul.NewConsul(kv, config.Registry.Prefix)

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{
		consul, config.Verbose, config.Debug,
	}
	http.HandleFunc("/events", forwarderHandler.Handle)

	log.Printf(`listening on "%s"`, config.Web.Listen)
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
