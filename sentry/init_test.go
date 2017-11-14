package sentry

import (
	"testing"

	"github.com/evalphobia/logrus_sentry"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_ShouldFailForInvalidDSN(t *testing.T) {
	err := Init(Config{
		DSN: "£££",
	})
	assert.Error(t, err)
}

func TestInit_ShouldRegisterLogrusSentryHookForValidDSN(t *testing.T) {
	err := Init(Config{
		DSN:   "http://login:password@localhost/marathon-consul",
		Level: "panic",
	})
	require.NoError(t, err)

	stdLog := log.StandardLogger()

	require.NotEmpty(t, stdLog.Hooks[log.PanicLevel])
	assert.IsType(t, &logrus_sentry.SentryHook{}, stdLog.Hooks[log.PanicLevel][0])
}
