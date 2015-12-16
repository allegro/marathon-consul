package marathon

import (
	"fmt"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/tasks"
)

type MarathonerStub struct {
	AppsStub  []*apps.App
	AppStub   map[tasks.AppId]*apps.App
	TasksStub map[tasks.AppId][]*tasks.Task
}

func (m MarathonerStub) Apps() ([]*apps.App, error) {
	return m.AppsStub, nil
}

func (m MarathonerStub) App(id tasks.AppId) (*apps.App, error) {
	if app, ok := m.AppStub[id]; ok {
		return app, nil
	} else {
		return nil, fmt.Errorf("app not found")
	}
}

func (m MarathonerStub) Tasks(appId tasks.AppId) ([]*tasks.Task, error) {
	if app, ok := m.TasksStub[appId]; ok {
		return app, nil
	} else {
		return nil, fmt.Errorf("app not found")
	}
}

func MarathonerStubForApps(args ...*apps.App) *MarathonerStub {
	appsMap := make(map[tasks.AppId]*apps.App)
	tasksMap := make(map[tasks.AppId][]*tasks.Task)

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
