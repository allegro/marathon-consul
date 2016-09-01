package consul

import (
	"testing"
	"github.com/allegro/marathon-consul/apps"
	"github.com/stretchr/testify/assert"
)

func TestServiceName_WithSeparator(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID: "/rootGroup/subGroup/subSubGroup/name",
	}

	// when
	serviceName := serviceName(app, ".")

	// then
	assert.Equal(t, "rootGroup.subGroup.subSubGroup.name", serviceName)
}

func TestServiceName_WithEmptyConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": ""},
	}

	// when
	serviceName := serviceName(app, "-")

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", serviceName)
}

func TestServiceName_WithConsulLabelSetToTrue(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "true"},
	}

	// when
	serviceName := serviceName(app, "-")

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", serviceName)
}

func TestServiceName_WithCustomConsulLabelEscapingChars(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "/some-other/name"},
	}

	// when
	serviceName := serviceName(app, "-")

	// then
	assert.Equal(t, "some-other-name", serviceName)
}

func TestServiceName_WithInvalidLabelValue(t *testing.T) {
	t.Parallel()

	// given
	app := &apps.App{
		ID:     "/rootGroup/subGroup/subSubGroup/name",
		Labels: map[string]string{"consul": "     ///"},
	}

	// when
	serviceName := serviceName(app, "-")

	// then
	assert.Equal(t, "rootGroup-subGroup-subSubGroup-name", serviceName)
}
