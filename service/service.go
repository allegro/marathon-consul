package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/allegro/marathon-consul/apps"
)

type ID string

func (id ID) String() string {
	return string(id)
}

type Service struct {
	ID                ID
	Name              string
	Tags              []string
	AgentAddress      string
	EnableTagOverride bool
}

func (s *Service) TaskID() (apps.TaskID, error) {
	for _, tag := range s.Tags {
		if strings.HasPrefix(tag, "marathon-task:") {
			return apps.TaskID(strings.TrimPrefix(tag, "marathon-task:")), nil
		}
	}
	return apps.TaskID(""), errors.New("marathon-task tag missing")
}

func MarathonTaskTag(taskID apps.TaskID) string {
	return fmt.Sprintf("marathon-task:%s", taskID)
}

type Registry interface {
	GetAllServices() ([]*Service, error)
	GetServices(name string) ([]*Service, error)
	Register(task *apps.Task, app *apps.App) error
	DeregisterByTask(taskID apps.TaskID) error
	Deregister(toDeregister *Service) error
}
