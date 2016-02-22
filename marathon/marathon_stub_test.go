package marathon_test

import (
	"testing"

	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
)

func TestMarathonStub(t *testing.T) {
	t.Parallel()
	// given
	m := marathon.MarathonerStubWithLeaderForApps("some.host:1234", utils.ConsulApp("/test/app", 3))
	// when
	leader, _ := m.Leader()
	// then
	assert.Equal(t, "some.host:1234", leader)
	// when
	apps, _ := m.ConsulApps()
	// then
	assert.Len(t, apps, 1)
	// when
	existingApp, _ := m.App("/test/app")
	// then
	assert.NotNil(t, existingApp)
	//when
	notExistingApp, errOnNotExistingApp := m.App("/not/existing/app")
	// then
	assert.Error(t, errOnNotExistingApp)
	assert.Nil(t, notExistingApp)
	// when
	existingTasks, _ := m.Tasks("/test/app")
	// then
	assert.Len(t, existingTasks, 3)
	// when
	notExistingTasks, errOnNotExistingTasks := m.Tasks("/not/existing/app")
	// then
	assert.Error(t, errOnNotExistingTasks)
	assert.Nil(t, notExistingTasks)
}
