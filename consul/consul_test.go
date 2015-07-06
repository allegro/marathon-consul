package consul

import (
	"bytes"
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/mocks"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testApp   = &apps.App{ID: "testApp"}
	testTask  = &tasks.Task{ID: "testTask", AppID: "testApp", Host: "test"}
	appPrefix = "marathon"
)

func TestSyncApps(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()

	deleteMe := &api.KVPair{Key: "marathon/deleteMe", Value: []byte("app")}
	deleteMeTask := &api.KVPair{Key: "marathon/deleteMe/tasks/test", Value: []byte("task")}
	testAppKV := &api.KVPair{Key: "marathon/testApp", Value: []byte("app")}
	testAppKVTask := &api.KVPair{Key: "marathon/testApp/tasks/test", Value: []byte("task")}

	kv.Put(deleteMe)
	kv.Put(deleteMeTask)
	kv.Put(testAppKV)
	kv.Put(testAppKVTask)

	// test!
	consul := Consul{kv, appPrefix}
	err := consul.SyncApps([]*apps.App{testApp})
	assert.Nil(t, err)

	// testApp should have been updated
	newTestApp, _, err := kv.Get(testAppKV.Key)
	assert.Nil(t, err)
	assert.NotEqual(t, testAppKV, newTestApp)

	// deleteMe should have been deleted
	newDeleteMe, _, err := kv.Get(deleteMe.Key)
	assert.Nil(t, err)
	assert.Nil(t, newDeleteMe)

	// deleteMeTask should have been deleted
	newDeleteMeTask, _, err := kv.Get(deleteMeTask.Key)
	assert.Nil(t, err)
	assert.Nil(t, newDeleteMeTask)
}

func TestUpdateApp(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()

	oldAppKV := &api.KVPair{Key: "marathon/testApp", Value: []byte("app")}
	kv.Put(oldAppKV)

	// test!
	consul := Consul{kv, appPrefix}
	err := consul.UpdateApp(testApp)
	assert.Nil(t, err)

	// testApp should have been updated
	newAppKV, _, err := kv.Get(oldAppKV.Key)
	assert.Nil(t, err)
	assert.NotEqual(t, oldAppKV, newAppKV)
}

func TestDeleteApp(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()
	oldAppKV := testApp.KV()
	oldAppKV.Key = WithPrefix(appPrefix, oldAppKV.Key)
	kv.Put(oldAppKV)

	// test!
	consul := Consul{kv, appPrefix}
	err := consul.DeleteApp(testApp)
	assert.Nil(t, err)

	// testApp should have been deleted
	newAppKV, _, err := kv.Get(oldAppKV.Key)
	assert.Nil(t, err)
	assert.Nil(t, newAppKV)
}

func TestSyncTasks(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()

	testAppKV := &api.KVPair{Key: "marathon/testApp", Value: []byte("app")}
	deleteTaskKV := &api.KVPair{Key: "marathon/testApp/tasks/delete", Value: []byte("task")}
	kv.Put(testAppKV)
	kv.Put(deleteTaskKV)

	tasks := []*tasks.Task{testTask}

	// test!
	consul := Consul{kv, appPrefix}
	err := consul.SyncTasks(testApp.ID, tasks)
	assert.Nil(t, err)

	// deleteTaskKV should not be present
	newDeleteTaskKV, _, err := kv.Get(deleteTaskKV.Key)
	assert.Nil(t, err)
	assert.Nil(t, newDeleteTaskKV)

	// a new task (with ID "new" should have been added)
	newTask, _, err := kv.Get("marathon/testApp/tasks/testTask")
	if assert.NotNil(t, newTask) {
		assert.True(t, bytes.Equal(newTask.Value, testTask.KV().Value))
	}
}

func TestUpdateTask(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()
	oldTaskKV := &api.KVPair{
		Key:   "marathon/testApp/tasks/testTask",
		Value: []byte(""),
	}
	kv.Put(oldTaskKV)

	// test!
	consul := Consul{kv, appPrefix}
	err := consul.UpdateTask(testTask)
	assert.Nil(t, err)

	// testTask should have been updated
	newTaskKV, _, err := kv.Get(oldTaskKV.Key)
	assert.Nil(t, err)
	assert.NotEqual(t, oldTaskKV, newTaskKV)
}

func TestCreateTask(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()

	// test!
	consul := Consul{kv, appPrefix}
	err := consul.UpdateTask(testTask)
	assert.Nil(t, err)

	// testTask should have been updated
	newTaskKV, _, err := kv.Get("marathon/testApp/tasks/testTask")
	assert.Nil(t, err)
	assert.NotNil(t, newTaskKV)
}

func TestDeleteTask(t *testing.T) {
	t.Parallel()

	kv := mocks.NewKVer()
	oldTaskKV := &api.KVPair{
		Key:   "marathon/testApp/tasks/testTask",
		Value: []byte(""),
	}
	kv.Put(oldTaskKV)

	// test!
	consul := Consul{kv, appPrefix}
	err := consul.DeleteTask(testTask)
	assert.Nil(t, err)

	// testTask should have been deleted
	newTaskKV, _, err := kv.Get(oldTaskKV.Key)
	assert.Nil(t, err)
	assert.Nil(t, newTaskKV)
}
