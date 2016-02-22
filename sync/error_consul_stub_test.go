package sync

import "fmt"

import (
	"github.com/allegro/marathon-consul/apps"
	consulapi "github.com/hashicorp/consul/api"
)

type errorConsul struct {
}

func (c errorConsul) GetServices(name string) ([]*consulapi.CatalogService, error) {
	return nil, fmt.Errorf("Error occured")
}

func (c errorConsul) GetAllServices() ([]*consulapi.CatalogService, error) {
	return nil, fmt.Errorf("Error occured")
}

func (c errorConsul) Register(task *apps.Task, app *apps.App) error {
	return fmt.Errorf("Error occured")
}

func (c errorConsul) Deregister(serviceId apps.TaskId, agent string) error {
	return fmt.Errorf("Error occured")
}

func (c errorConsul) GetAgent(agent string) (*consulapi.Client, error) {
	return nil, fmt.Errorf("Error occured")
}
