package consul

import (
	"fmt"
	"sync/atomic"

	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

type Agent struct {
	Client        *consulapi.Client
	IP            string
	failures      uint32
	failureMetric string
}

func (a *Agent) IncFailures() uint32 {
	return atomic.AddUint32(&a.failures, 1)
}

func (a *Agent) ClearFailures() {
	atomic.StoreUint32(&a.failures, 0)
}

func (a *ConcurrentAgents) createAgent(ipAddress string) (*Agent, error) {
	client, err := a.newConsulClient(ipAddress)
	agent := &Agent{
		Client: client,
		IP:     ipAddress,
	}
	return agent, err
}

func (a *ConcurrentAgents) newConsulClient(ipAddress string) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()

	config.HttpClient = a.client

	config.Address = fmt.Sprintf("%s:%s", ipAddress, a.config.Port)

	if a.config.Token != "" {
		config.Token = a.config.Token
	}

	if a.config.SslEnabled {
		config.Scheme = "https"
	}

	if a.config.Auth.Enabled {
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: a.config.Auth.Username,
			Password: a.config.Auth.Password,
		}
	}

	log.WithFields(log.Fields{
		"Address":                config.Address,
		"Scheme":                 config.Scheme,
		"Timeout":                config.HttpClient.Timeout,
		"BasicAuthEnabled":       a.config.Auth.Enabled,
		"TokenEnabled":           a.config.Token != "",
		"SslVerificationEnabled": a.config.SslVerify,
	}).Debug("Creating Consul client")

	return consulapi.NewClient(config)
}
