package consul

import (
	"testing"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
)

func TestConsulStub(t *testing.T) {
	t.Parallel()
	// given
	consul := NewConsulStub()
	app := utils.ConsulApp("test", 3)
	services, err := consul.GetAllServices()
	assert.NoError(t, err)
	testServices, err := consul.GetServices("test")
	assert.NoError(t, err)

	// then
	assert.Empty(t, services)
	assert.Empty(t, testServices)

	// when
	for _, task := range app.Tasks {
		err = consul.Register(&task, app)
		assert.NoError(t, err)
	}
	services, err = consul.GetAllServices()
	assert.NoError(t, err)
	testServices, err = consul.GetServices("test")
	assert.NoError(t, err)

	// then
	assert.Len(t, services, 3)
	assert.Len(t, testServices, 3)

	// when
	err = consul.DeregisterByTask(app.Tasks[1].ID)
	services, _ = consul.GetAllServices()
	taskIds := consul.RegisteredTaskIds()

	// then
	assert.NoError(t, err)
	assert.Len(t, services, 2)
	assert.Contains(t, taskIds, apps.TaskId("test.0"))
	assert.Contains(t, taskIds, apps.TaskId("test.2"))

	// given
	consul.FailDeregisterByTaskForId(app.Tasks[0].ID)

	// when
	err = consul.DeregisterByTask(app.Tasks[0].ID)

	// then
	assert.Error(t, err)

	// given
	consul.FailRegisterForId(app.Tasks[0].ID)

	// when
	err = consul.Register(&app.Tasks[0], app)

	// then
	assert.Error(t, err)

	// when
	err = consul.DeregisterByTask(app.Tasks[2].ID)

	// then
	assert.NoError(t, err)
	assert.Len(t, consul.RegisteredTaskIds(), 1)

	// when
	app = utils.ConsulApp("other", 2)
	for _, task := range app.Tasks {
		consul.Register(&task, app)
	}
	services, _ = consul.GetAllServices()
	testServices, _ = consul.GetServices("test")
	otherServices, _ := consul.GetServices("other")

	// then
	assert.Len(t, services, 3)
	assert.Len(t, testServices, 1)
	assert.Len(t, otherServices, 2)
}
