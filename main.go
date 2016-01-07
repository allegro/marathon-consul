package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/config"
	service "github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/sync"
	"github.com/allegro/marathon-consul/web"
	"net/http"
)

var VERSION string

func main() {

	log.WithField("Version", VERSION).Info("Starting marathon-consul")

	config, err := config.New()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = metrics.Init(config.Metrics)
	if err != nil {
		log.Fatal(err.Error())
	}

	service := service.New(config.Consul)
	remote, err := marathon.New(config.Marathon)
	if err != nil {
		log.Fatal(err.Error())
	}

	sync := sync.New(config.Sync, remote, service)
	go sync.StartSyncServicesJob()

	// set up routes
	http.HandleFunc("/health", web.HealthHandler)
	http.HandleFunc("/events", web.NewEventHandler(service, remote).Handle)

	log.WithField("Port", config.Web.Listen).Info("Listening")
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
