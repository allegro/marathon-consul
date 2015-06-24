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
	// TODO: sync tasks

	return nil
}
