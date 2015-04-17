package main

import (
	"flag"
	"github.com/hashicorp/consul/api"
	"log"
	"net/http"
	"net/url"
)

var (
	parallelism = flag.Int("parallelism", 4, "set this many keys at once (per request)")
	consul      = flag.String("consul", "http://localhost:8500", "Consul location")
	token       = flag.String("token", "", "Consul ACL token")
	datacenter  = flag.String("datacenter", "", "Consul datacenter")
	user        = flag.String("user", "", "Consul basic auth username")
	pass        = flag.String("pass", "", "Consul basic auth password")
	serve       = flag.String("serve", ":8000", "accept connections at this address")
)

func main() {
	flag.Parse()

	url, err := url.Parse(*consul)
	if err != nil {
		log.Fatal(err.Error())
	}

	var auth *api.HttpBasicAuth
	if *user != "" && *pass != "" {
		auth = &api.HttpBasicAuth{
			Username: *user,
			Password: *pass,
		}
	} else {
		auth = nil
	}

	forwarder, err := NewForwarder(
		&api.Config{
			Address:    url.Host,
			Scheme:     url.Scheme,
			Datacenter: *datacenter,
			HttpAuth:   auth,
			Token:      *token,
		},
		*parallelism,
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{*forwarder}
	http.HandleFunc("/events", forwarderHandler.Handle)

	// TODO: register

	log.Printf(`listening on "%s"`, *serve)
	log.Fatal(http.ListenAndServe(*serve, nil))
}
