package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAuth(t *testing.T) {
	t.Parallel()

	// null case
	reg := Registry{}
	auth, err := reg.GetAuth()

	assert.Nil(t, auth)
	assert.Nil(t, err)

	// good case
	reg = Registry{Auth: "a:b"}
	auth, err = reg.GetAuth()

	assert.Nil(t, err)
	if assert.NotNil(t, auth) {
		assert.Equal(t, "a", auth.Username)
		assert.Equal(t, "b", auth.Password)
	}

	// bad case
	reg = Registry{Auth: "a"}
	auth, err = reg.GetAuth()

	assert.Nil(t, auth)
	assert.Equal(t, ErrBadCredentials, err)
}

func TestRegistryConfigURL(t *testing.T) {
	t.Parallel()

	reg := Registry{Location: "x"}
	c, err := reg.Config()
	assert.Nil(t, c)
	assert.Equal(t, err, ErrNoScheme)
}
