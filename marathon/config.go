package marathon

import (
	log "github.com/Sirupsen/logrus"
	"strings"
)

type Config struct {
	Location string
	Protocol string
	Username string
	Password string
}

func (m Config) Validate() {
	m.Protocol = strings.ToLower(m.Protocol)
	if !(m.Protocol == "http" || m.Protocol == "https") {
		log.WithField("protocol", m.Protocol).Fatal("invalid protocol")
	}
}
