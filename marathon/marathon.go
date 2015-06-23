package marathon

import (
	"github.com/CiscoCloud/marathon-consul/apps"
	"net/url"
)

type Marathoner interface {
	Apps() ([]*apps.App, error)
}

type Marathon struct {
	Location string
	Protocol string
	Auth     *url.Userinfo
}

func NewMarathon(location, protocol string, auth *url.Userinfo) (*Marathon, error) {
	return &Marathon{location, protocol, auth}, nil
}
