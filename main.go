package main

import (
	"github.com/CiscoCloud/marathon-consul/config"
	service "github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/marathon"
	"github.com/CiscoCloud/marathon-consul/metrics"
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
	// TODO: sync should run continuously with some time interval
	sync := marathon.NewMarathonSync(remote, service)
	go sync.SyncServices()

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{service, remote}
	http.HandleFunc("/events", forwarderHandler.Handle)

	log.WithField("port", config.Web.Listen).Info("listening")
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
