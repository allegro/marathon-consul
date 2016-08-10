package sync

import (
	"errors"

	"github.com/allegro/marathon-consul/apps"
	consulapi "github.com/hashicorp/consul/api"
)

type errorConsul struct {
}

func (c errorConsul) GetServices(name string) ([]*consulapi.CatalogService, error) {
	return nil, errors.New("Error occured")
}

func (c errorConsul) GetAllServices() ([]*consulapi.CatalogService, error) {
	return nil, errors.New("Error occured")
}

func (c errorConsul) Register(task *apps.Task, app *apps.App) error {
	return errors.New("Error occured")
}

func (c errorConsul) Deregister(serviceID apps.TaskID, agent string) error {
	return errors.New("Error occured")
}

func (c errorConsul) ServiceName(app *apps.App) string {
	return ""
}

func (c errorConsul) GetAgent(agent string) (*consulapi.Client, error) {
	return nil, errors.New("Error occured")
}
