package sync

import (
	"errors"

	"github.com/allegro/marathon-consul/apps"
)

type errorMarathon struct {
}

func (m errorMarathon) ConsulApps() ([]*apps.App, error) {
	return nil, errors.New("Error")
}

func (m errorMarathon) App(id apps.AppID) (*apps.App, error) {
	return nil, errors.New("Error")
}

func (m errorMarathon) Tasks(appID apps.AppID) ([]apps.Task, error) {
	return nil, errors.New("Error")
}

func (m errorMarathon) Leader() (string, error) {
	return "", errors.New("Error")
}
