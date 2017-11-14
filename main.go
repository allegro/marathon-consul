package main

import (
	"net/http"

	"github.com/allegro/marathon-consul/config"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/sentry"
	"github.com/allegro/marathon-consul/sse"
	"github.com/allegro/marathon-consul/sync"
	"github.com/allegro/marathon-consul/web"
	log "github.com/sirupsen/logrus"
)

var VERSION string

func main() {
	log.WithField("Version", VERSION).Info("Starting marathon-consul")

	config, err := config.New()
	if err != nil {
		log.Fatal(err.Error())
	}

	config.Log.Sentry.Release = VERSION
	if sentryErr := sentry.Init(config.Log.Sentry); sentryErr != nil {
		log.Fatal(sentryErr)
	}

	err = metrics.Init(config.Metrics)
	if err != nil {
		log.Fatal(err.Error())
	}

	consulInstance := consul.New(config.Consul)
	// TODO(tz) - move Leader from sync module to highest level config, access like config.Leader
	remote, err := marathon.New(config.Marathon)
	if err != nil {
		log.Fatal(err.Error())
	}

	sync.New(config.Sync, remote, consulInstance, consulInstance.AddAgentsFromApps).StartSyncServicesJob()

	//TODO: Use context instead of stop function.
	var stopSSE sse.Stop
	go func() {
		stopSSE, err = sse.NewHandler(config.SSE, config.Web, remote, consulInstance)
		if err != nil {
			log.WithError(err).Fatal("Cannot instantiate SSE handler")
		}
	}()
	defer stopSSE()

	http.HandleFunc("/health", web.HealthHandler)

	log.WithField("Port", config.Web.Listen).Info("Listening")
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
