package config

import (
	"github.com/CiscoCloud/marathon-consul/marathon"
	log "github.com/Sirupsen/logrus"
	"net/url"
	"strings"
)

type MarathonConfig struct {
	Location string
	Protocol string
	Username string
	Password string
}

func (m MarathonConfig) Validate() {
	// protocol
	m.Protocol = strings.ToLower(m.Protocol)
	if !(m.Protocol == "http" || m.Protocol == "https") {
		log.WithField("protocol", m.Protocol).Fatal("invalid protocol")
	}
}

func (m MarathonConfig) NewMarathon() (marathon.Marathon, error) {
	m.Validate()

	return marathon.NewMarathon(
		m.Location,
		m.Protocol,
		url.UserPassword(m.Username, m.Password),
	)
}
