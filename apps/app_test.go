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

func TestRegistrationIntent_NameWithSeparator(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: "/rootGroup/subGroup/subSubGroup/name",
	}

	// when
	intent := app.RegistrationIntents(".")[0]

	// then
	assert.Equal(t, "rootGroup.subGroup.subSubGroup.name", intent.Name)
}

func TestRegistrationIntent_NameWithEmptyConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": ""},
	}

	// when
	intent := app.RegistrationIntents("-")[0]

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", intent.Name)
}

func TestRegistrationIntent_NameWithConsulLabelSetToTrue(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "true"},
	}

	// when
	intent := app.RegistrationIntents("-")[0]

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", intent.Name)
}

func TestRegistrationIntent_NameWithCustomConsulLabelEscapingChars(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "/some-other/name"},
	}

	// when
	intent := app.RegistrationIntents("-")[0]

	// then
	assert.Equal(t, "some-other-name", intent.Name)
}

func TestRegistrationIntent_NameWithInvalidLabelValue(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "     ///"},
	}

	// when
	intent := app.RegistrationIntents("-")[0]

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", intent.Name)
}

func TestRegistrationIntent_PickFirstPort(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: "name",
	}

	// when
	intent := app.RegistrationIntents("-")[0]

	// then
	assert.Equal(t, 0, intent.PortIndex)
}

func TestRegistrationIntent_WithTags(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: 	"name",
		Labels: map[string]string{"private": "tag", "other": "irrelevant"},
	}

	// when
	intent := app.RegistrationIntents("-")[0]

	// then
	assert.Equal(t, []string{"private"}, intent.Tags)
}

func TestRegistrationIntent_NoOverrideViaPortDefinitionsIfNoConsulLabelThere(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: 	"app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"other": "tag"},
			},
			PortDefinition{
			},
		},
	}

	// when
	intents := app.RegistrationIntents("-")

	// then
	assert.Len(t, intents, 1)
	assert.Equal(t, "app-name", intents[0].Name)
	assert.Equal(t, 0, intents[0].PortIndex)
	assert.Equal(t, []string{"private"}, intents[0].Tags)
}

func TestRegistrationIntent_OverrideNameAndAddTagsViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: 	"app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
				Labels: map[string]string{"consul": "other-name", "other": "tag"},
			},
			PortDefinition{
			},
		},
	}

	// when
	intents := app.RegistrationIntents("-")

	// then
	assert.Len(t, intents, 1)
	assert.Equal(t, "other-name", intents[0].Name)
	assert.Equal(t, 0, intents[0].PortIndex)
	assert.Equal(t, []string{"private", "other"}, intents[0].Tags)
}

func TestRegistrationIntent_PickDifferentPortViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: 	"app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			PortDefinition{
			},
			PortDefinition{
				Labels: map[string]string{"consul": "true"},
			},
		},
	}

	// when
	intent := app.RegistrationIntents("-")[0]

	// then
	assert.Equal(t, 1, intent.PortIndex)
}

func TestRegistrationIntent_MultipleIntentsViaPortDefinitionIfMultipleContainConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID: 	"app-name",
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

	// when
	intents := app.RegistrationIntents("-")

	// then
	assert.Len(t, intents, 2)
	assert.Equal(t, "first-name", intents[0].Name)
	assert.Equal(t, 0, intents[0].PortIndex)
	assert.Equal(t, []string{"common-tag", "first-tag"}, intents[0].Tags)
	assert.Equal(t, "second-name", intents[1].Name)
	assert.Equal(t, 1, intents[1].PortIndex)
	assert.Equal(t, []string{"common-tag", "second-tag"}, intents[1].Tags)
}
