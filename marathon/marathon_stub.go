package marathon

import (
	"fmt"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/tasks"
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
	if app, ok := m.AppStub[id]; ok {
		return app, nil
	} else {
		return nil, fmt.Errorf("app not found")
	}
}

func (m MarathonerStub) Tasks(appId string) ([]*tasks.Task, error) {
	if app, ok := m.TasksStub[appId]; ok {
		return app, nil
	} else {
		return nil, fmt.Errorf("app not found")
	}
}

func MarathonerStubForApps(args ...*apps.App) *MarathonerStub {
	appsMap := make(map[string]*apps.App)
	tasksMap := make(map[string][]*tasks.Task)

	for _, app := range args {
		appsMap[app.ID] = app
		tasks := []*tasks.Task{}
		for _, task := range app.Tasks {
			t := task
			tasks = append(tasks, &t)
		}
		tasksMap[app.ID] = tasks
	}

	return &MarathonerStub{
		AppsStub:  args,
		AppStub:   appsMap,
		TasksStub: tasksMap,
	}
}
