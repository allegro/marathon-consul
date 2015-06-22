package config

import (
	"github.com/CiscoCloud/marathon-consul/marathon"
	log "github.com/Sirupsen/logrus"
	"strings"
)

type MarathonConfig struct {
	Location string
	Protocol string
	Auth     string
}

func (m MarathonConfig) Validate() {
	// protocol
	m.Protocol = strings.ToLower(m.Protocol)
	if !(m.Protocol == "http" || m.Protocol == "https") {
		log.WithField("protocol", m.Protocol).Fatal("invalid protocol")
	}

	// auth
	if m.Auth != "" && !strings.Contains(m.Auth, ":") {
		log.Fatal("invalid auth")
	}
}

func (m MarathonConfig) NewMarathon() (*marathon.Marathon, error) {
	m.Validate()

	return marathon.NewMarathon(
		m.Location,
		m.Protocol,
		m.Auth,
	)
}
