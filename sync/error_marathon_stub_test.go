package sync

import (
	"fmt"

	"github.com/allegro/marathon-consul/apps"
)

type errorMarathon struct {
}

func (m errorMarathon) ConsulApps() ([]*apps.App, error) {
	return nil, fmt.Errorf("Error")
}

func (m errorMarathon) App(id apps.AppId) (*apps.App, error) {
	return nil, fmt.Errorf("Error")
}

func (m errorMarathon) Tasks(appId apps.AppId) ([]*apps.Task, error) {
	return nil, fmt.Errorf("Error")
}

func (m errorMarathon) Leader() (string, error) {
	return "", fmt.Errorf("Error")
}
