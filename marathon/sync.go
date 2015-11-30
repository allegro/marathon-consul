package marathon

import (
	"github.com/CiscoCloud/marathon-consul/consul"
	log "github.com/Sirupsen/logrus"
)

type MarathonSync struct {
	marathon Marathoner
	consul   consul.Consul
}

func NewMarathonSync(marathon Marathoner, consul consul.Consul) *MarathonSync {
	return &MarathonSync{marathon, consul}
}

func (m *MarathonSync) Sync() error {

	//	TODO: Register and unregister services

	// apps
	log.Info("syncing apps")
	apps, err := m.marathon.Apps()
	if err != nil {
		return err
	}
	err = m.consul.SyncApps(apps)
	if err != nil {
		return err
	}

	// tasks
	log.Info("syncing tasks")
	for _, app := range apps {
		log.WithField("app", app.ID).Debug("syncing tasks for app")
		tasks, err := m.marathon.Tasks(app.ID)
		if err != nil {
			return err
		}
		err = m.consul.SyncTasks(app.ID, tasks)
		if err != nil {
			return err
		}
	}

	return nil
}
