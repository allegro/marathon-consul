package config

import (
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/sync"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfig_NewReturnsErrorWhenFileNotExist(t *testing.T) {
	os.Args = []string{"./marathon-consul", "--config-file=unknown.json"}
	_, err := New()
	assert.Error(t, err)
}

func TestConfig_NewReturnsErrorWhenFileIsNotJson(t *testing.T) {
	os.Args = []string{"./marathon-consul", "--config-file=config.go"}
	_, err := New()
	assert.Error(t, err)
}

func TestConfig_ShouldBeMergedWithFileDefaultsAndFlags(t *testing.T) {

	expected := &Config{
		Consul: consul.ConsulConfig{
			Auth: consul.Auth{Enabled: false,
				Username: "",
				Password: ""},
			Port:       "8500",
			SslEnabled: false,
			SslVerify:  true,
			SslCert:    "",
			SslCaCert:  "",
			Token:      ""},
		Web:  struct{ Listen string }{Listen: ":4000"},
		Sync: sync.Config{Interval: 91},
		Marathon: marathon.Config{Location: "marathon.host:8080",
			Protocol:  "https",
			Username:  "user",
			Password:  "pass",
			VerifySsl: false},
		Metrics: metrics.Config{Target: "stdout",
			Prefix:   "default",
			Interval: 31,
			Addr:     ""},
		LogLevel:   "debug",
		configFile: "config.json",
	}

	os.Args = []string{"./marathon-consul", "--log-level=debug", "--config-file=config.json", "--marathon-location=localhost:8080"}
	actual, err := New()

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
