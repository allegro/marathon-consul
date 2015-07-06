// package consul deals with syncing Marathon apps and tasks to Consul.
package consul

import (
	"bytes"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"strings"
)

type Consul struct {
	kv         KVer
	AppsPrefix string
}

func NewConsul(kv KVer, prefix string) Consul {
	return Consul{kv, prefix}
}

// SyncApps takes a *complete* list of apps from Marathon and compares them
// against the apps in Consul. It performs any necessary updates, then
// recursively deletes any apps that are present in Consul but not the given
// list.
func (consul *Consul) SyncApps(apps []*apps.App) error {
	remoteKeys, _, err := consul.kv.List(consul.AppsPrefix)
	if err != nil {
		return err
	}
	remotePairs := MapKVPairs(remoteKeys)
	localPairs := MapApps(apps)

	// add/update any new apps
	for _, local := range localPairs {
		// make sure local apps have prefix
		local.Key = WithPrefix(consul.AppsPrefix, local.Key)

		remote, exists := remotePairs[local.Key]
		if !exists || !bytes.Equal(local.Value, remote.Value) {
			_, err := consul.kv.Put(local)
			if err != nil {
				return err
			}
		}
	}

	// remove any outdated apps
	for _, remote := range remotePairs {
		// we deal with tasks in SyncTasks, we don't need to touch them here.
		if strings.Contains(remote.Key, "tasks") {
			continue
		}

		if _, exists := localPairs[WithoutPrefix(consul.AppsPrefix, remote.Key)]; !exists {
			_, err := consul.kv.Delete(remote.Key)
			if err != nil {
				return err
			}

			// delete tasks
			appTasks, _, err := consul.kv.List(remote.Key)
			if err != nil {
				return err
			}

			for _, task := range appTasks {
				_, err := consul.kv.Delete(task.Key)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// UpdateApp takes an App and updates it in Consul
func (consul *Consul) UpdateApp(app *apps.App) error {
	var err error = nil

	local := app.KV()
	local.Key = WithPrefix(consul.AppsPrefix, local.Key)

	remote, _, err := consul.kv.Get(local.Key)
	if err != nil {
		return err
	}

	if remote == nil || len(remote.Value) == 0 || !bytes.Equal(local.Value, remote.Value) {
		_, err = consul.kv.Put(local)
	}

	return err
}

// DeleteApp takes an App and deletes it from Consul
func (consul *Consul) DeleteApp(app *apps.App) error {
	_, err := consul.kv.Delete(WithPrefix(consul.AppsPrefix, app.Key()))
	return err
}

// SyncTasks takes a *complete* list of tasks from a Marathon App and compares
// them against the tasks in Consul. It performs any necessary updates, then
// deletes any tasks that are present in Consul but not the list.
func (consul *Consul) SyncTasks(appId string, tasks []*tasks.Task) error {
	// remove prefix from app ID if present
	if appId[0] == '/' {
		appId = appId[1:]
	}

	remoteKeys, _, err := consul.kv.List(fmt.Sprintf(
		"%s/%s/tasks", consul.AppsPrefix, appId,
	))
	if err != nil {
		return err
	}

	remotePairs := MapKVPairs(remoteKeys)
	localPairs := MapTasks(tasks)

	// add/update any new tasks
	for _, local := range localPairs {
		// make sure local pairs have prefix
		local.Key = WithPrefix(consul.AppsPrefix, local.Key)

		remote, exists := remotePairs[local.Key]
		if !exists || !bytes.Equal(local.Value, remote.Value) {
			_, err := consul.kv.Put(local)
			if err != nil {
				return err
			}
		}
	}

	// remove any outdated tasks
	for _, remote := range remotePairs {
		if _, exists := localPairs[WithoutPrefix(consul.AppsPrefix, remote.Key)]; !exists {
			_, err := consul.kv.Delete(remote.Key)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// UpdateTask takes a Task and updates it in Consul
func (consul *Consul) UpdateTask(task *tasks.Task) error {
	local := task.KV()
	local.Key = WithPrefix(consul.AppsPrefix, local.Key)

	// we always want to update tasks
	_, err := consul.kv.Put(local)
	return err
}

// DeleteTask taske a Task and deletes it from Consul
func (consul *Consul) DeleteTask(task *tasks.Task) error {
	_, err := consul.kv.Delete(WithPrefix(consul.AppsPrefix, task.Key()))
	return err
}
