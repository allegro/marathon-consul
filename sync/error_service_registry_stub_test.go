package sync

import (
	"errors"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/service"
)

type errorServiceRegistry struct {
}

func (c errorServiceRegistry) GetServices(name string) ([]*service.Service, error) {
	return nil, errors.New("Error occured")
}

func (c errorServiceRegistry) GetAllServices() ([]*service.Service, error) {
	return nil, errors.New("Error occured")
}

func (c errorServiceRegistry) Register(task *apps.Task, app *apps.App) error {
	return errors.New("Error occured")
}

func (c errorServiceRegistry) DeregisterByTask(taskID apps.TaskID) error {
	return errors.New("Error occured")
}

func (c errorServiceRegistry) Deregister(toDeregister *service.Service) error {
	return errors.New("Error occured")
}

func (c errorServiceRegistry) ServiceNames(app *apps.App) []string {
	return []string{}
}
