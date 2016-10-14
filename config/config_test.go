package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/sync"
	"github.com/allegro/marathon-consul/web"
	"github.com/stretchr/testify/assert"
)

func TestConfig_NewReturnsErrorWhenFileNotExist(t *testing.T) {
	clear()

	// given
	os.Args = []string{"./marathon-consul", "--config-file=unknown.json"}

	// when
	_, err := New()

	// then
	assert.Error(t, err)
}

func TestConfig_NewReturnsErrorWhenFileIsNotJson(t *testing.T) {
	clear()

	// given
	os.Args = []string{"./marathon-consul", "--config-file=config.go"}

	// when
	_, err := New()

	// then
	assert.Error(t, err)
}

func TestConfig_ShouldReturnErrorForBadLogLevel(t *testing.T) {
	clear()

	// given
	os.Args = []string{"./marathon-consul", "--log-level=bad"}

	// when
	_, err := New()

	// then
	assert.Error(t, err)
}

func TestConfig_ShouldParseFlags(t *testing.T) {
	clear()

	// given
	os.Args = []string{"./marathon-consul", "--log-level=debug", "--marathon-location=test.host:8080", "--log-format=json"}

	// when
	actual, err := New()

	// then
	assert.NoError(t, err)
	assert.Equal(t, "debug", actual.Log.Level)
	assert.Equal(t, "json", actual.Log.Format)
	assert.Equal(t, "test.host:8080", actual.Marathon.Location)
}

func TestConfig_ShouldUseTextFormatterWhenFormatterIsUnknown(t *testing.T) {
	clear()

	// given
	os.Args = []string{"./marathon-consul", "--log-level=debug", "--log-format=unknown"}

	// when
	_, err := New()

	// then
	assert.NoError(t, err)
}

func TestConfig_ShouldBeMergedWithFileDefaultsAndFlags(t *testing.T) {
	clear()
	expected := &Config{
		Consul: consul.Config{
			Auth: consul.Auth{Enabled: false,
				Username: "",
				Password: ""},
			Port:                   "8500",
			SslEnabled:             false,
			SslVerify:              true,
			SslCert:                "",
			SslCaCert:              "",
			Token:                  "",
			Tag:                    "marathon",
			Timeout:                3 * time.Second,
			RequestRetries:         5,
			AgentFailuresTolerance: 3,
			ConsulNameSeparator:    ".",
		},
		Web: web.Config{
			Listen:       ":4000",
			QueueSize:    1000,
			WorkersCount: 10,
		},
		Sync: sync.Config{
			Interval: 15 * time.Minute,
			Enabled:  true,
			Leader:   "",
			Force:    false,
		},
		Marathon: marathon.Config{Location: "localhost:8080",
			Protocol:  "http",
			Username:  "",
			Password:  "",
			VerifySsl: true,
			Timeout:   30 * time.Second},
		Metrics: metrics.Config{Target: "stdout",
			Prefix:   "default",
			Interval: 30 * time.Second,
			Addr:     ""},
		Log: struct{ Level, Format, File string }{
			Level:  "info",
			Format: "text",
			File:   "",
		},
		configFile: "../debian/config.json",
	}

	os.Args = []string{"./marathon-consul", "--log-level=debug", "--config-file=../debian/config.json", "--marathon-location=localhost:8080"}
	actual, err := New()

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

// http://stackoverflow.com/a/29169727/1387612
func clear() {
	p := reflect.ValueOf(config).Elem()
	p.Set(reflect.Zero(p.Type()))
}
