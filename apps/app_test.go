package apps

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseApps(t *testing.T) {
	t.Parallel()

	appBlob, _ := ioutil.ReadFile("apps.json")

	expected := []*App{
		{
			HealthChecks: []HealthCheck{
				{
					Path:                   "/",
					PortIndex:              0,
					Protocol:               "HTTP",
					GracePeriodSeconds:     5,
					IntervalSeconds:        20,
					TimeoutSeconds:         20,
					MaxConsecutiveFailures: 3,
				},
			},
			ID: "/bridged-webapp",
			Tasks: []Task{
				{
					ID:                 "test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8",
					AppID:              "/test",
					Host:               "192.168.2.114",
					Ports:              []int{31315},
					HealthCheckResults: []HealthCheckResult{{Alive: true}},
				},
				{
					ID:    "test.4453212c-1a81-11e5-bdb6-e6cb6734eaf8",
					AppID: "/test",
					Host:  "192.168.2.114",
					Ports: []int{31797},
				},
			},
		},
	}
	apps, err := ParseApps(appBlob)
	assert.NoError(t, err)
	assert.Len(t, apps, 1)
	assert.Equal(t, expected, apps)
}

func TestParseApp(t *testing.T) {
	t.Parallel()

	appBlob, _ := ioutil.ReadFile("app.json")

	expected := &App{Labels: map[string]string{"consul": "true", "public": "tag"},
		HealthChecks: []HealthCheck{{Path: "/",
			PortIndex:              0,
			Protocol:               "HTTP",
			GracePeriodSeconds:     10,
			IntervalSeconds:        5,
			TimeoutSeconds:         10,
			MaxConsecutiveFailures: 3}},
		ID: "/myapp",
		Tasks: []Task{{
			ID:    "myapp.cc49ccc1-9812-11e5-a06e-56847afe9799",
			AppID: "/myapp",
			Host:  "10.141.141.10",
			Ports: []int{31678,
				31679,
				31680,
				31681},
			HealthCheckResults: []HealthCheckResult{{Alive: true}}},
			{
				ID:    "myapp.c8b449f0-9812-11e5-a06e-56847afe9799",
				AppID: "/myapp",
				Host:  "10.141.141.10",
				Ports: []int{31307,
					31308,
					31309,
					31310},
				HealthCheckResults: []HealthCheckResult{{Alive: true}}}}}

	app, err := ParseApp(appBlob)
	assert.NoError(t, err)
	assert.Equal(t, expected, app)
}

func TestConsulApp(t *testing.T) {
	t.Parallel()

	// when
	app := &App{
		Labels: map[string]string{"consul": "true"},
	}

	// then
	assert.True(t, app.IsConsulApp())

	// when
	app = &App{
		Labels: map[string]string{"consul": "someName", "marathon": "true"},
	}

	// then
	assert.True(t, app.IsConsulApp())

	// when
	app = &App{
		Labels: map[string]string{},
	}

	// then
	assert.False(t, app.IsConsulApp())
}

func TestAppId_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "appId", AppId("appId").String())
}

var dummyTask = &Task{
	ID:    TaskId("some-task"),
	Ports: []int{1337},
}

func TestRegistrationIntent_NameWithoutConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: "/rootGroup/subGroup/subSubGroup/name",
	}

	// when
	intent := app.RegistrationIntents(dummyTask, ".")[0]

	// then
	assert.Equal(t, "rootGroup.subGroup.subSubGroup.name", intent.Name)
}

func TestRegistrationIntent_Name(t *testing.T) {
	t.Parallel()

	var intentNameTestsData = []struct {
		consulLabel  string
		expectedName string
	}{
		{"", "rootGroup-subGroup-subSubGroup-name"},
		{"true", "rootGroup-subGroup-subSubGroup-name"},
		{"/some-other/name", "some-other-name"},
		{"     ///", "rootGroup-subGroup-subSubGroup-name"},
	}

	for _, testData := range intentNameTestsData {
		// given
		app := &App{
			ID:     "/rootGroup/subGroup/subSubGroup/name",
			Labels: map[string]string{"consul": testData.consulLabel},
		}

		// when
		intent := app.RegistrationIntents(dummyTask, "-")[0]

		// then
		if intent.Name != testData.expectedName {
			t.Errorf("Registration name from consul label '%s' was '%s', expected '%s'", testData.consulLabel, intent.Name, testData.expectedName)
		}
	}
}

func TestRegistrationIntent_PickFirstPort(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: "name",
	}
	task := &Task{
		Ports: []int{1234, 5678},
	}

	// when
	intent := app.RegistrationIntents(task, "-")[0]

	// then
	assert.Equal(t, 1234, intent.Port)
}

func TestRegistrationIntent_WithTags(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "name",
		Labels: map[string]string{"private": "tag", "other": "irrelevant"},
	}

	// when
	intent := app.RegistrationIntents(dummyTask, "-")[0]

	// then
	assert.Equal(t, []string{"private"}, intent.Tags)
}

func TestRegistrationIntent_NoOverrideViaPortDefinitionsIfNoConsulLabelThere(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"other": "tag"},
			},
			PortDefinition{},
		},
	}
	task := &Task{
		Ports: []int{1234, 5678},
	}

	// when
	intents := app.RegistrationIntents(task, "-")

	// then
	assert.Len(t, intents, 1)
	assert.Equal(t, "app-name", intents[0].Name)
	assert.Equal(t, 1234, intents[0].Port)
	assert.Equal(t, []string{"private"}, intents[0].Tags)
}

func TestRegistrationIntent_OverrideNameAndAddTagsViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "other-name", "other": "tag"},
			},
			PortDefinition{},
		},
	}
	task := &Task{
		Ports: []int{1234, 5678},
	}

	// when
	intents := app.RegistrationIntents(task, "-")

	// then
	assert.Len(t, intents, 1)
	assert.Equal(t, "other-name", intents[0].Name)
	assert.Equal(t, 1234, intents[0].Port)
	assert.Equal(t, []string{"private", "other"}, intents[0].Tags)
}

func TestRegistrationIntent_PickDifferentPortViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{},
			PortDefinition{
				Labels: map[string]string{"consul": "true"},
			},
		},
	}
	task := &Task{
		Ports: []int{1234, 5678},
	}

	// when
	intent := app.RegistrationIntents(task, "-")[0]

	// then
	assert.Equal(t, 5678, intent.Port)
}

func TestRegistrationIntent_PickExplicitPortViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Port:   1337,
				Labels: map[string]string{"consul": "true"},
			},
		},
	}
	task := &Task{
		Ports: []int{},
	}

	// when
	intent := app.RegistrationIntents(task, "-")[0]

	// then
	assert.Equal(t, 1337, intent.Port)
}

func TestRegistrationIntent_MultipleIntentsViaPortDefinitionIfMultipleContainConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "common-tag": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "first-name", "first-tag": "tag"},
			},
			PortDefinition{
				Labels: map[string]string{"consul": "second-name", "second-tag": "tag"},
			},
		},
	}
	task := &Task{
		Ports: []int{1234, 5678},
	}

	// when
	intents := app.RegistrationIntents(task, "-")

	// then
	assert.Len(t, intents, 2)
	assert.Equal(t, "first-name", intents[0].Name)
	assert.Equal(t, 1234, intents[0].Port)
	assert.Equal(t, []string{"common-tag", "first-tag"}, intents[0].Tags)
	assert.Equal(t, "second-name", intents[1].Name)
	assert.Equal(t, 5678, intents[1].Port)
	assert.Equal(t, []string{"common-tag", "second-tag"}, intents[1].Tags)
}

func TestHasSameConsulNamesAs_SameConfigsWithoutPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
	}

	// expect
	assert.True(t, app.HasSameConsulNamesAs(app))
}

func TestHasSameConsulNamesAs_DifferentConfigsSameNameWithoutPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
	}
	other := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "app-name"},
	}

	// expect
	assert.True(t, app.HasSameConsulNamesAs(other))
}

func TestHasSameConsulNamesAs_DifferentConfigsWithoutPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "some-name"},
	}
	other := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "different-name"},
	}

	// expect
	assert.False(t, app.HasSameConsulNamesAs(other))
}

func TestHasSameConsulNamesAs_SameConfigsWithPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "true"},
			},
		},
	}

	// expect
	assert.True(t, app.HasSameConsulNamesAs(app))
}

func TestHasSameConsulNamesAs_DifferentConfigsSameNamesWithPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "true"},
			},
		},
	}
	other := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "app-name"},
			},
		},
	}

	// expect
	assert.True(t, app.HasSameConsulNamesAs(other))
}

func TestHasSameConsulNamesAs_DifferentConfigsDifferentNamesWithPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "some-name"},
			},
		},
	}
	other := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "other-name"},
			},
		},
	}

	// expect
	assert.False(t, app.HasSameConsulNamesAs(other))
}

func TestHasSameConsulNamesAs_DifferentConfigsDifferentNumberOfRegitrationsWithPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "some-name"},
			},
		},
	}
	other := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "some-name"},
			},
			PortDefinition{
				Labels: map[string]string{"consul": "yet-another-name"},
			},
		},
	}

	// expect
	assert.False(t, app.HasSameConsulNamesAs(other))
}
