package consul

import (
	"fmt"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConsulStub(t *testing.T) {
	t.Parallel()
	// given
	labels := map[string]string{
		"consul": "true",
		"public": "tag",
	}
	healthChecks := []apps.HealthCheck{
		apps.HealthCheck{
			Path:                   "/",
			Protocol:               "HTTP",
			PortIndex:              0,
			IntervalSeconds:        60,
			TimeoutSeconds:         20,
			MaxConsecutiveFailures: 3,
		},
	}
	consul := NewConsulStub()
	app := utils.ConsulApp("test", 3)
	stubError := fmt.Errorf("Some error")
	services, err := consul.GetAllServices()
	// then
	assert.Empty(t, services)
	assert.NoError(t, err)
	// when
	for _, task := range app.Tasks {
		consul.Register(MarathonTaskToConsulService(task, app.HealthChecks, app.Labels))
	}
	services, _ = consul.GetAllServices()
	// then
	assert.Len(t, services, 3)
	// when
	err = consul.Deregister(app.Tasks[1].ID, "")
	services, _ = consul.GetAllServices()
	servicesIds := consul.RegisteredServicesIds()
	// then
	assert.NoError(t, err)
	assert.Len(t, services, 2)
	assert.Contains(t, servicesIds, "test.0")
	assert.Contains(t, servicesIds, "test.2")
	// given
	consul.ErrorServices[app.Tasks[0].ID] = stubError
	// when
	err = consul.Deregister(app.Tasks[0].ID, "")
	// then
	assert.Equal(t, stubError, err)
	// when
	err = consul.Register(MarathonTaskToConsulService(app.Tasks[0], healthChecks, labels))
	// then
	assert.Equal(t, stubError, err)
	// when
	err = consul.Deregister(app.Tasks[2].ID, "")
	// then
	assert.NoError(t, err)
	assert.Len(t, consul.RegisteredServicesIds(), 1)
}
