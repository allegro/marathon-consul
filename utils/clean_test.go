package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCleanID(t *testing.T) {
	assert.Equal(
		t,
		"nice-app-id",
		CleanID("/nice/app/id"),
	)
}
