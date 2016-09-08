package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/allegro/marathon-consul/apps"
)

type ServiceId string

func (id ServiceId) String() string {
	return string(id)
}

type Service struct {
	ID                      ServiceId
	Name                    string
	Tags                    []string
	RegisteringAgentAddress string
}

func (s *Service) TaskId() (apps.TaskID, error) {
	for _, tag := range s.Tags {
		if strings.HasPrefix(tag, "marathon-task:") {
			return apps.TaskID(strings.TrimPrefix(tag, "marathon-task:")), nil
		}
	}
	return apps.TaskID(""), errors.New("marathon-task tag missing")
}

func MarathonTaskTag(taskId apps.TaskID) string {
	return fmt.Sprintf("marathon-task:%s", taskId)
}

type ServiceRegistry interface {
	GetAllServices() ([]*Service, error)
	GetServices(name string) ([]*Service, error)
	Register(task *apps.Task, app *apps.App) error
	DeregisterByTask(taskId apps.TaskID) error
	Deregister(toDeregister *Service) error
	ServiceNames(app *apps.App) []string
}
