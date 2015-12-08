package consul

import (
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/tasks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsTaskHealthy(t *testing.T) {
	t.Parallel()
	assert.False(t, IsTaskHealthy(nil))
	assert.False(t, IsTaskHealthy([]tasks.HealthCheckResult{}))
	assert.False(t, IsTaskHealthy([]tasks.HealthCheckResult{
		tasks.HealthCheckResult{Alive: false},
	}))
	assert.False(t, IsTaskHealthy([]tasks.HealthCheckResult{
		tasks.HealthCheckResult{Alive: true},
		tasks.HealthCheckResult{Alive: false},
	}))
	assert.True(t, IsTaskHealthy([]tasks.HealthCheckResult{
		tasks.HealthCheckResult{Alive: true},
		tasks.HealthCheckResult{Alive: true},
	}))
}

func TestMarathonTaskToConsulServiceMapping_WithNoHttpChecks(t *testing.T) {
	t.Parallel()

	// given
	task := tasks.Task{
		ID:    "someTask",
		AppID: "someApp",
		Host:  "127.0.0.6",
		Ports: []int{8090, 8443},
	}

	labels := map[string]string{
		"consul": "true",
		"public": "tag",
	}
	healthChecks := []apps.HealthCheck{
		apps.HealthCheck{
			Path:                   "/",
			Protocol:               "TCP",
			PortIndex:              0,
			IntervalSeconds:        60,
			TimeoutSeconds:         20,
			MaxConsecutiveFailures: 3,
		},
	}

	// when
	service := MarathonTaskToConsulService(task, healthChecks, labels)

	// then
	assert.Equal(t, "127.0.0.6", service.Address)
	assert.Equal(t, 8090, service.Port)
	assert.Nil(t, service.Check)
	assert.Empty(t, service.Checks)
}

func TestMarathonTaskToConsulServiceMapping(t *testing.T) {
	t.Parallel()

	// given
	task := tasks.Task{
		ID:    "someTask",
		AppID: "someApp",
		Host:  "127.0.0.6",
		Ports: []int{8090, 8443},
	}

	labels := map[string]string{
		"consul": "true",
		"public": "tag",
	}
	healthChecks := []apps.HealthCheck{
		apps.HealthCheck{
			Path:                   "/api/health",
			Protocol:               "HTTP",
			PortIndex:              0,
			IntervalSeconds:        60,
			TimeoutSeconds:         20,
			MaxConsecutiveFailures: 3,
		},
	}

	// when
	service := MarathonTaskToConsulService(task, healthChecks, labels)

	// then
	assert.Equal(t, "127.0.0.6", service.Address)
	assert.Equal(t, []string{"marathon", "public"}, service.Tags)
	assert.Equal(t, 8090, service.Port)
	assert.NotNil(t, "http://127.0.0.6:8090/api/health", service.Check)
	assert.Empty(t, service.Checks)
	assert.Equal(t, "http://127.0.0.6:8090/api/health", service.Check.HTTP)
	assert.Equal(t, "60s", service.Check.Interval)
}
