package service

import (
	"errors"
	"strings"

	"github.com/allegro/marathon-consul/apps"
	"fmt"
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

func (s *Service) TaskId() (apps.TaskId, error) {
	for _, tag := range s.Tags {
		if strings.HasPrefix(tag, "marathon-task:") {
			return apps.TaskId(strings.TrimPrefix(tag, "marathon-task:")), nil
		}
	}
	return apps.TaskId(""), errors.New("marathon-task tag missing")
}

func MarathonTaskTag(taskId apps.TaskId) string {
	return fmt.Sprintf("marathon-task:%s", taskId)
}

type ServiceRegistry interface {
	GetAllServices() ([]*Service, error)
	GetServices(name string) ([]*Service, error)
	Register(task *apps.Task, app *apps.App) error
	DeregisterByTask(taskId apps.TaskId, agentAddress string) error
	Deregister(toDeregister *Service) error
	ServiceName(app *apps.App) string
}
