package utils

import (
	"github.com/allegro/marathon-consul/apps"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsulApp(t *testing.T) {
	t.Parallel()
	expected := &apps.App{Labels: map[string]string{"consul": "true"},
		HealthChecks: []apps.HealthCheck(nil),
		ID:           "name",
		Tasks: []apps.Task{{ID: "name.0",
			TaskStatus:         "",
			AppID:              "name",
			Host:               "localhost",
			Ports:              []int{8080},
			HealthCheckResults: []apps.HealthCheckResult{{Alive: true}}},
			{ID: "name.1",
				TaskStatus:         "",
				AppID:              "name",
				Host:               "localhost",
				Ports:              []int{8081},
				HealthCheckResults: []apps.HealthCheckResult{{Alive: true}}}}}

	app := ConsulApp("name", 2)
	assert.Equal(t, expected, app)
}

func TestNonConsulApp(t *testing.T) {
	t.Parallel()
	expected := &apps.App{Labels: map[string]string{},
		HealthChecks: []apps.HealthCheck(nil),
		ID:           "name",
		Tasks: []apps.Task{{ID: "name.0",
			TaskStatus:         "",
			AppID:              "name",
			Host:               "localhost",
			Ports:              []int{8080},
			HealthCheckResults: []apps.HealthCheckResult{{Alive: true}}},
			{ID: "name.1",
				TaskStatus:         "",
				AppID:              "name",
				Host:               "localhost",
				Ports:              []int{8081},
				HealthCheckResults: []apps.HealthCheckResult{{Alive: true}}}}}

	app := NonConsulApp("name", 2)
	assert.Equal(t, expected, app)
}

func TestConsulAppWithUnhelathyInstancesgreaterThanInstances(t *testing.T) {
	t.Parallel()
	expected := &apps.App{Labels: map[string]string{"consul": "true"},
		HealthChecks: []apps.HealthCheck(nil),
		ID:           "name",
		Tasks: []apps.Task{{ID: "name.0",
			TaskStatus:         "",
			AppID:              "name",
			Host:               "localhost",
			Ports:              []int{8080},
			HealthCheckResults: nil},
			{ID: "name.1",
				TaskStatus:         "",
				AppID:              "name",
				Host:               "localhost",
				Ports:              []int{8081},
				HealthCheckResults: nil}}}

	app := ConsulAppWithUnhealthyInstances("name", 2, 5)
	assert.Equal(t, expected, app)
}

func TestConsulAppWithUnhelathyInstances(t *testing.T) {
	t.Parallel()
	expected := &apps.App{Labels: map[string]string{"consul": "true"},
		HealthChecks: []apps.HealthCheck(nil),
		ID:           "name",
		Tasks: []apps.Task{{ID: "name.0",
			TaskStatus:         "",
			AppID:              "name",
			Host:               "localhost",
			Ports:              []int{8080},
			HealthCheckResults: nil},
			{ID: "name.1",
				TaskStatus:         "",
				AppID:              "name",
				Host:               "localhost",
				Ports:              []int{8081},
				HealthCheckResults: []apps.HealthCheckResult{{Alive: true}}}}}

	app := ConsulAppWithUnhealthyInstances("name", 2, 1)
	assert.Equal(t, expected, app)
}
