package consul

import (
	"testing"
	"github.com/allegro/marathon-consul/apps"
	"github.com/stretchr/testify/assert"
)

var dummyTask = &apps.Task{
	ID:    apps.TaskId("some-task"),
	Ports: []int{1337},
}

func TestRegistrationIntent_NameWithSeparator(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: "/rootGroup/subGroup/subSubGroup/name",
	}

	// when
	intent := toRegistrationIntent(dummyTask, app, ".")

	// then
	assert.Equal(t, "rootGroup.subGroup.subSubGroup.name", intent.Name)
}

func TestRegistrationIntent_NameWithEmptyConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": ""},
	}

	// when
	intent := toRegistrationIntent(dummyTask, app, "-")

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", intent.Name)
}

func TestRegistrationIntent_NameWithConsulLabelSetToTrue(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "true"},
	}

	// when
	intent := toRegistrationIntent(dummyTask, app, "-")

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", intent.Name)
}

func TestRegistrationIntent_NameWithCustomConsulLabelEscapingChars(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "/some-other/name"},
	}

	// when
	intent := toRegistrationIntent(dummyTask, app, "-")

	// then
	assert.Equal(t, "some-other-name", intent.Name)
}

func TestRegistrationIntent_NameWithInvalidLabelValue(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "     ///"},
	}

	// when
	intent := toRegistrationIntent(dummyTask, app, "-")

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", intent.Name)
}

func TestRegistrationIntent_PickFirstPort(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: "name",
	}
	task := &apps.Task{
		Ports: []int{1234, 5678},
	}

	// when
	intent := toRegistrationIntent(task, app, "-")

	// then
	assert.Equal(t, 1234, intent.Port)
}

func TestRegistrationIntent_WithTags(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: 	"name",
		Labels: map[string]string{"private": "tag", "other": "irrelevant"},
	}

	// when
	intent := toRegistrationIntent(dummyTask, app, "-")

	// then
	assert.Equal(t, []string{"private"}, intent.Tags)
}

func TestRegistrationIntent_NoOverrideViaPortDefinitionsIfNoConsulLabelThere(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: 	"app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []apps.PortDefinition{
			apps.PortDefinition{
				Labels: map[string]string{"other": "tag"},
			},
			apps.PortDefinition{
			},
		},
	}
	task := &apps.Task{
		Ports: []int{1234, 5678},
	}

	// when
	intent := toRegistrationIntent(task, app, "-")

	// then
	assert.Equal(t, "app-name", intent.Name)
	assert.Equal(t, 1234, intent.Port)
	assert.Equal(t, []string{"private"}, intent.Tags)
}

func TestRegistrationIntent_OverrideNameAndAddTagsViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: 	"app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []apps.PortDefinition{
			apps.PortDefinition{
				Labels: map[string]string{"consul": "other-name", "other": "tag"},
			},
			apps.PortDefinition{
			},
		},
	}
	task := &apps.Task{
		Ports: []int{1234, 5678},
	}

	// when
	intent := toRegistrationIntent(task, app, "-")

	// then
	assert.Equal(t, "other-name", intent.Name)
	assert.Equal(t, 1234, intent.Port)
	assert.Equal(t, []string{"private", "other"}, intent.Tags)
}

func TestRegistrationIntent_PickDifferentPortViaPortDefinitions(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: 	"app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []apps.PortDefinition{
			apps.PortDefinition{
			},
			apps.PortDefinition{
				Labels: map[string]string{"consul": "true"},
			},
		},
	}
	task := &apps.Task{
		Ports: []int{1234, 5678},
	}

	// when
	intent := toRegistrationIntent(task, app, "-")

	// then
	assert.Equal(t, 5678, intent.Port)
}

func TestRegistrationIntent_PickFirstMatchingPortDefinitionIfMultipleContainConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: 	"app-name",
		Labels: map[string]string{"consul": "true", "private": "tag"},
		PortDefinitions: []apps.PortDefinition{
			apps.PortDefinition{
				Labels: map[string]string{"consul": "first"},
			},
			apps.PortDefinition{
				Labels: map[string]string{"consul": "second"},
			},
		},
	}
	task := &apps.Task{
		Ports: []int{1234, 5678},
	}

	// when
	intent := toRegistrationIntent(task, app, "-")

	// then
	assert.Equal(t, "first", intent.Name)
}
