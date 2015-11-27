package main

import (
	"github.com/CiscoCloud/marathon-consul/config"
	"github.com/CiscoCloud/marathon-consul/consul"
	service "github.com/CiscoCloud/marathon-consul/consul-services"
	"github.com/CiscoCloud/marathon-consul/marathon"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"github.com/ogier/pflag"
)

const Name = "marathon-consul"
const Version = "0.2.0"

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
	service.AddCmdFlags(pflag.NewFlagSet("marathon-consul", pflag.ContinueOnError))
	service := *service.New()

	// set up initial sync
	remote, err := config.Marathon.NewMarathon()
	if err != nil {
		log.Fatal(err.Error())
	}
	sync := marathon.NewMarathonSync(remote, consul)
	go sync.Sync()

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{consul, service, remote}
	http.HandleFunc("/events", forwarderHandler.Handle)

	log.WithField("port", config.Web.Listen).Info("listening")
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
