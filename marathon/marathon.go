package marathon

import (
	"github.com/CiscoCloud/marathon-consul/apps"
)

type Marathoner interface {
	Apps() ([]*apps.App, error)
}

type Marathon struct {
	Location string
	Protocol string
	Auth     string
}

func NewMarathon(location, protocol, auth string) (*Marathon, error) {
	return nil, nil
}
