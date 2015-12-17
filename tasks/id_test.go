package tasks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestId_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "id", Id("id").String())
}

func TestAppId_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "appId", AppId("appId").String())
}

func TestAppId_ConsulServiceName(t *testing.T) {
	t.Parallel()
	// given
	id := AppId("/rootGroup/subGroup/subSubGroup/name")

	// when
	serviceName := id.ConsulServiceName()

	// then
	assert.Equal(t, "rootGroup.subGroup.subSubGroup.name", serviceName)
}
