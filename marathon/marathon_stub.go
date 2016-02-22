package marathon

import (
	"fmt"

	"github.com/allegro/marathon-consul/apps"
)

type MarathonerStub struct {
	AppsStub  []*apps.App
	AppStub   map[apps.AppId]*apps.App
	TasksStub map[apps.AppId][]*apps.Task
	leader    string
}

func (m MarathonerStub) ConsulApps() ([]*apps.App, error) {
	return m.AppsStub, nil
}

func (m MarathonerStub) App(id apps.AppId) (*apps.App, error) {
	if app, ok := m.AppStub[id]; ok {
		return app, nil
	} else {
		return nil, fmt.Errorf("app not found")
	}
}

func (m MarathonerStub) Tasks(appId apps.AppId) ([]*apps.Task, error) {
	if app, ok := m.TasksStub[appId]; ok {
		return app, nil
	} else {
		return nil, fmt.Errorf("app not found")
	}
}

func (m MarathonerStub) Leader() (string, error) {
	return m.leader, nil
}

func MarathonerStubWithLeaderForApps(leader string, args ...*apps.App) *MarathonerStub {
	stub := MarathonerStubForApps(args...)
	stub.leader = leader
	return stub
}

func MarathonerStubForApps(args ...*apps.App) *MarathonerStub {
	appsMap := make(map[apps.AppId]*apps.App)
	tasksMap := make(map[apps.AppId][]*apps.Task)

	for _, app := range args {
		appsMap[app.ID] = app
		tasks := []*apps.Task{}
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
		leader:    "localhost:8080",
	}
}
