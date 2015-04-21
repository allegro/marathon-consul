package main

import (
	"crypto/tls"
	"flag"
	"github.com/hashicorp/consul/api"
	"log"
	"net/http"
	"net/url"
)

var (
	consul      = flag.String("consul", "http://localhost:8500", "Consul location")
	datacenter  = flag.String("datacenter", "", "Consul datacenter")
	noverify    = flag.Bool("noverify", false, "don't verify certificates")
	parallelism = flag.Int("parallelism", 4, "set this many keys at once (per request)")
	pass        = flag.String("pass", "", "Consul basic auth password")
	serve       = flag.String("serve", ":8000", "accept connections at this address")
	token       = flag.String("token", "", "Consul ACL token")
	user        = flag.String("user", "", "Consul basic auth username")
	verbose     = flag.Bool("verbose", false, "enable verbose logging")
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
			HttpClient: &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: *noverify,
					},
				},
			},
		},
		*parallelism,
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	forwarder.Verbose = *verbose

	// set up routes
	http.HandleFunc("/health", HealthHandler)
	forwarderHandler := &ForwardHandler{*forwarder}
	http.HandleFunc("/events", forwarderHandler.Handle)

	// TODO: register

	log.Printf(`listening on "%s"`, *serve)
	log.Fatal(http.ListenAndServe(*serve, nil))
}
