package sync

import (
	"errors"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/marathon"
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

func (m errorMarathon) EventStream([]string, int, int) (*marathon.Streamer, error) {
	return &marathon.Streamer{}, errors.New("Error")
}

func (m errorMarathon) AmILeader() (bool, error) {
	return false, errors.New("Error")
}
