package marathon

import (
	"errors"

	"github.com/allegro/marathon-consul/apps"
	"sync"
)

type MarathonerStub struct {
	AppsStub       []*apps.App
	AppStub        map[apps.AppID]*apps.App
	TasksStub      map[apps.AppID][]*apps.Task
	leader         string
	interactionsMu sync.RWMutex
	interactions   bool
}

func (m *MarathonerStub) ConsulApps() ([]*apps.App, error) {
	m.noteInteraction()
	return m.AppsStub, nil
}

func (m *MarathonerStub) App(id apps.AppID) (*apps.App, error) {
	m.noteInteraction()
	if app, ok := m.AppStub[id]; ok {
		return app, nil
	}
	return nil, errors.New("app not found")
}

func (m *MarathonerStub) Tasks(appID apps.AppID) ([]*apps.Task, error) {
	m.noteInteraction()
	if app, ok := m.TasksStub[appID]; ok {
		return app, nil
	}
	return nil, errors.New("app not found")
}

func (m *MarathonerStub) Leader() (string, error) {
	m.noteInteraction()
	return m.leader, nil
}

func (m *MarathonerStub) Interactions() bool {
	m.interactionsMu.RLock()
	defer m.interactionsMu.RUnlock()
	return m.interactions
}

func (m *MarathonerStub) noteInteraction() {
	m.interactionsMu.Lock()
	defer m.interactionsMu.Unlock()
	m.interactions = true
}

func MarathonerStubWithLeaderForApps(leader string, args ...*apps.App) *MarathonerStub {
	stub := MarathonerStubForApps(args...)
	stub.leader = leader
	return stub
}

func MarathonerStubForApps(args ...*apps.App) *MarathonerStub {
	appsMap := make(map[apps.AppID]*apps.App)
	tasksMap := make(map[apps.AppID][]*apps.Task)

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
