package apps

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseApps(t *testing.T) {
	t.Parallel()

	appBlob, _ := ioutil.ReadFile("testdata/apps.json")

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

func TestAppInt(t *testing.T) {
	appBlob, _ := ioutil.ReadFile("testdata/lorem-ipsum.json")
	app, err := ParseApp(appBlob)
	assert.NoError(t, err)
	t.Log(app)

	assert.Equal(t, 2, app.RegistrationIntentsNumber())

	task := Task{Ports: []int{0, 1, 2, 3}}
	intents := app.RegistrationIntents(&task, ".")

	assert.Contains(t, intents[0].Tags, "Lorem ipsum dolor sit amet, consectetur adipiscing elit")
	assert.Contains(t, intents[1].Tags, "secureConnection:true")
	assert.NotContains(t, intents[0].Tags, "secureConnection:true")
	assert.NotContains(t, intents[1].Tags, "Lorem ipsum dolor sit amet, consectetur adipiscing elit")
}

func TestParseApp(t *testing.T) {
	t.Parallel()

	appBlob, _ := ioutil.ReadFile("testdata/app.json")

	expected := &App{Labels: map[string]string{"consul": "true", "public": "tag"},
		HealthChecks: []HealthCheck{
			{
				Path:                   "/",
				PortIndex:              0,
				Protocol:               "HTTP",
				GracePeriodSeconds:     10,
				IntervalSeconds:        5,
				TimeoutSeconds:         10,
				MaxConsecutiveFailures: 3,
			},
			{
				Path:                   "/custom",
				Port:                   8123,
				Protocol:               "HTTP",
				GracePeriodSeconds:     10,
				IntervalSeconds:        5,
				TimeoutSeconds:         10,
				MaxConsecutiveFailures: 3,
			},
		},
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
	assert.Equal(t, "appId", AppID("appId").String())
}

var dummyTask = &Task{
	ID:    TaskID("some-task"),
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
			{
				Labels: map[string]string{"other": "tag"},
			},
			{},
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

func TestRegistrationIntent_DontPanicIfTaskHasNoPorts(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			{
				Labels: map[string]string{"other": "tag"},
			},
			{},
		},
	}
	task := &Task{
		Ports: []int{},
	}

	// when
	intents := app.RegistrationIntents(task, "-")

	// then
	assert.Empty(t, intents)
}

func TestRegistrationIntent_OverrideNameAndAddTagsViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			{
				Labels: map[string]string{"consul": "other-name", "other": "tag"},
			},
			{},
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
	assert.Equal(t, []string{"other", "private"}, intents[0].Tags)
}

func TestRegistrationIntent_PickDifferentPortViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []PortDefinition{
			{},
			{
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

func TestRegistrationIntent_MultipleIntentsViaPortDefinitionIfMultipleContainConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "common-tag": "tag"},
		PortDefinitions: []PortDefinition{
			{
				Labels: map[string]string{"consul": "first-name", "first-tag": "tag"},
			},
			{
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
	assert.Equal(t, []string{"first-tag", "common-tag"}, intents[0].Tags)
	assert.Equal(t, "second-name", intents[1].Name)
	assert.Equal(t, 5678, intents[1].Port)
	assert.Equal(t, []string{"second-tag", "common-tag"}, intents[1].Tags)
}

func TestRegistrationIntent_TaskHasLessPortsThanApp(t *testing.T) {
	t.Parallel()

	// given
	app := &App{
		ID:     "app-name",
		Labels: map[string]string{"consul": "true", "common-tag": "tag"},
		PortDefinitions: []PortDefinition{
			{
				Labels: map[string]string{"consul": "first-name", "first-tag": "tag"},
			},
			{
				Labels: map[string]string{"consul": "second-name", "second-tag": "tag"},
			},
		},
	}
	task := &Task{
		Ports: []int{1234},
	}

	// when
	intents := app.RegistrationIntents(task, "-")

	// then
	assert.Len(t, intents, 1)
	assert.Equal(t, "first-name", intents[0].Name)
	assert.Equal(t, 1234, intents[0].Port)
	assert.Equal(t, []string{"first-tag", "common-tag"}, intents[0].Tags)
}

func TestRegistrationIntentsNumber(t *testing.T) {
	for _, tc := range registrationIntentsNumberTestCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.given.RegistrationIntentsNumber())
		})
	}
}

var registrationIntentsNumberTestCases = []struct {
	name     string
	given    App
	expected int
}{
	{"Not a consul app", App{ID: "id"}, 0},
	{"No port definitions", App{
		ID:     "id",
		Labels: map[string]string{"consul": ""},
	}, 1},
	{"Single port definition", App{
		ID:     "id",
		Labels: map[string]string{"consul": ""},
		PortDefinitions: []PortDefinition{
			{
				Labels: map[string]string{"consul": ""},
			},
		},
	}, 1},
	{"Multiple port definitions", App{
		ID:     "id",
		Labels: map[string]string{"consul": ""},
		PortDefinitions: []PortDefinition{
			{
				Labels: map[string]string{"consul": ""},
			},
			{
				Labels: map[string]string{"consul": ""},
			},
		},
	}, 2},
}
