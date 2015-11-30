package main

import (
	"github.com/CiscoCloud/marathon-consul/config"
	service "github.com/CiscoCloud/marathon-consul/consul-services"
	"github.com/CiscoCloud/marathon-consul/marathon"
	log "github.com/Sirupsen/logrus"
	"github.com/ogier/pflag"
	"net/http"
)

const Name = "marathon-consul"
const Version = "0.2.0"

func main() {

	//	TODO: Handle command line flags
	service.AddCmdFlags(pflag.NewFlagSet("marathon-consul", pflag.ContinueOnError))
	service := *service.New()

	// set up initial sync
	//	TODO: Handle command line flags
	config := config.New()
	remote, err := config.Marathon.NewMarathon()
	if err != nil {
		log.Fatal(err.Error())
	}
	sync := marathon.NewMarathonSync(remote, service)
	go sync.SyncServices()

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{service, remote}
	http.HandleFunc("/events", forwarderHandler.Handle)

	log.WithField("port", config.Web.Listen).Info("listening")
	log.Fatal(http.ListenAndServe(config.Web.Listen, nil))
}
