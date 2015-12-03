package main

import (
	"github.com/CiscoCloud/marathon-consul/config"
	service "github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/marathon"
	"github.com/CiscoCloud/marathon-consul/metrics"
	"github.com/CiscoCloud/marathon-consul/sync"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

const Name = "marathon-consul"
const Version = "0.2.0"

func main() {

	config := config.New()

	metrics.Init(config.Metrics)

	service := service.New(config.Consul)
	remote, err := marathon.New(config.Marathon)

	if err != nil {
		log.Fatal(err.Error())
	}
	sync := sync.New(remote, service)
	go sync.StartSyncServicesJob(config.Sync.Interval)

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{service, remote}
	http.HandleFunc("/events", forwarderHandler.Handle)

	log.WithField("port", config.Web.Listen).Info("Listening")
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
