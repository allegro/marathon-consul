package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHostToIPv4_(t *testing.T) {
	t.Parallel()

	// given
	ip, err := HostToIPv4("2001:cdba:0000:0000:0000:0000:3257:9652")

	// then
	assert.Nil(t, ip)
	assert.Error(t, err)

	// when
	ip, err = HostToIPv4("127.0.0.1")

	// then
	assert.Equal(t, "127.0.0.1", ip.String())
	assert.NoError(t, err)

	// when
	ip, err = HostToIPv4("127.1.1.12")

	// then
	assert.Equal(t, "127.1.1.12", ip.String())
	assert.NoError(t, err)

	// when
	ip, err = HostToIPv4("localhost")

	// then
	assert.Equal(t, "127.0.0.1", ip.String())
	assert.NoError(t, err)

	// when
	ip, err = HostToIPv4("")

	// then
	assert.Nil(t, ip)
	assert.Error(t, err)
}
