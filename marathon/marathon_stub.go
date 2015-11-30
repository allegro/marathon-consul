package marathon

import (
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/tasks"
)

type MarathonerStub struct {
	AppsStub  []*apps.App
	AppStub   map[string]*apps.App
	TasksStub map[string][]*tasks.Task
}

func (m MarathonerStub) Apps() ([]*apps.App, error) {
	return m.AppsStub, nil
}

func (m MarathonerStub) App(id string) (*apps.App, error) {
	return m.AppStub[id], nil
}

func (m MarathonerStub) Tasks(appId string) ([]*tasks.Task, error) {
	return m.TasksStub[appId], nil
}
