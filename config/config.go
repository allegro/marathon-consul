package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"flag"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/sentry"
	"github.com/allegro/marathon-consul/sync"
	"github.com/allegro/marathon-consul/web"
)

type Config struct {
	Consul   consul.Config
	Web      web.Config
	Sync     sync.Config
	Marathon marathon.Config
	Metrics  metrics.Config
	Log      struct {
		Level  string
		Format string
		File   string
		Sentry sentry.Config
	}
	configFile string
}

var config = &Config{}

func New() (*Config, error) {
	flag.Parse()
	err := config.loadConfigFromFile()

	if err != nil {
		return nil, err
	}

	err = config.setLogOutput()
	if err != nil {
		return nil, err
	}
	config.setLogFormat()
	err = config.setLogLevel()
	if err != nil {
		return nil, err
	}

	return config, err
}

func init() {
	// Consul
	flag.StringVar(&config.Consul.Port, "consul-port", "8500", "Consul port")
	flag.BoolVar(&config.Consul.Auth.Enabled, "consul-auth", false, "Use Consul with authentication")
	flag.StringVar(&config.Consul.Auth.Username, "consul-auth-username", "", "The basic authentication username")
	flag.StringVar(&config.Consul.Auth.Password, "consul-auth-password", "", "The basic authentication password")
	flag.BoolVar(&config.Consul.SslEnabled, "consul-ssl", false, "Use HTTPS when talking to Consul")
	flag.BoolVar(&config.Consul.SslVerify, "consul-ssl-verify", true, "Verify certificates when connecting via SSL")
	flag.StringVar(&config.Consul.SslCert, "consul-ssl-cert", "", "Path to an SSL client certificate to use to authenticate to the Consul server")
	flag.StringVar(&config.Consul.SslCaCert, "consul-ssl-ca-cert", "", "Path to a CA certificate file, containing one or more CA certificates to use to validate the certificate sent by the Consul server to us")
	flag.StringVar(&config.Consul.Token, "consul-token", "", "The Consul ACL token")
	flag.StringVar(&config.Consul.Tag, "consul-tag", "marathon", "Common tag name added to every service registered in Consul, should be unique for every Marathon-cluster connected to Consul")
	flag.DurationVar(&config.Consul.Timeout.Duration, "consul-timeout", 3*time.Second, "Time limit for requests made by the Consul HTTP client. A Timeout of zero means no timeout")
	flag.Uint64Var(&config.Consul.AgentFailuresTolerance, "consul-max-agent-failures", 3, "Max number of consecutive request failures for agent before removal from cache")
	flag.Uint64Var(&config.Consul.RequestRetries, "consul-get-services-retry", 3, "Number of retries on failure when performing requests to Consul. Each retry uses different cached agent")
	flag.StringVar(&config.Consul.ConsulNameSeparator, "consul-name-separator", ".", "Separator used to create default service name for Consul")
	flag.StringVar(&config.Consul.IgnoredHealthChecks, "consul-ignored-healthchecks", "", "A comma separated blacklist of Marathon health check types that will not be migrated to Consul, e.g. command,tcp")

	// Web
	flag.StringVar(&config.Web.Listen, "listen", ":4000", "Accept connections at this address")
	flag.IntVar(&config.Web.QueueSize, "events-queue-size", 1000, "Size of events queue")
	flag.IntVar(&config.Web.WorkersCount, "workers-pool-size", 10, "Number of concurrent workers processing events")
	flag.Int64Var(&config.Web.MaxEventSize, "event-max-size", 4096, "Maximum size of event to process (bytes)")

	// Sync
	flag.BoolVar(&config.Sync.Enabled, "sync-enabled", true, "Enable Marathon-consul scheduled sync")
	flag.DurationVar(&config.Sync.Interval.Duration, "sync-interval", 15*time.Minute, "Marathon-consul sync interval")
	flag.StringVar(&config.Sync.Leader, "sync-leader", "", "Marathon cluster-wide node name (defaults to <hostname>:8080), the sync will run only if the specified node is the current Marathon-leader")
	flag.BoolVar(&config.Sync.Force, "sync-force", false, "Force leadership-independent Marathon-consul sync (run always)")

	// Marathon
	flag.StringVar(&config.Marathon.Location, "marathon-location", "localhost:8080", "Marathon URL")
	flag.StringVar(&config.Marathon.Protocol, "marathon-protocol", "http", "Marathon protocol (http or https)")
	flag.StringVar(&config.Marathon.Username, "marathon-username", "", "Marathon username for basic auth")
	flag.StringVar(&config.Marathon.Password, "marathon-password", "", "Marathon password for basic auth")
	flag.BoolVar(&config.Marathon.VerifySsl, "marathon-ssl-verify", true, "Verify certificates when connecting via SSL")
	flag.DurationVar(&config.Marathon.Timeout.Duration, "marathon-timeout", 30*time.Second, "Time limit for requests made by the Marathon HTTP client. A Timeout of zero means no timeout")

	// Metrics
	flag.StringVar(&config.Metrics.Target, "metrics-target", "stdout", "Metrics destination stdout or graphite (empty string disables metrics)")
	flag.StringVar(&config.Metrics.Prefix, "metrics-prefix", "default", "Metrics prefix (default is resolved to <hostname>.<app_name>")
	flag.DurationVar(&config.Metrics.Interval.Duration, "metrics-interval", 30*time.Second, "Metrics reporting interval")
	flag.StringVar(&config.Metrics.Addr, "metrics-location", "", "Graphite URL (used when metrics-target is set to graphite)")

	// Log
	flag.StringVar(&config.Log.Level, "log-level", "info", "Log level: panic, fatal, error, warn, info, or debug")
	flag.StringVar(&config.Log.Format, "log-format", "text", "Log format: JSON, text")
	flag.StringVar(&config.Log.File, "log-file", "", "Save logs to file (e.g.: `/var/log/marathon-consul.log`). If empty logs are published to STDERR")

	// Log -> Sentry
	flag.StringVar(&config.Log.Sentry.DSN, "sentry-dsn", "", "Sentry DSN. If it's not set sentry will be disabled")
	flag.StringVar(&config.Log.Sentry.Env, "sentry-env", "", "Sentry environment")
	flag.StringVar(&config.Log.Sentry.Level, "sentry-level", "error", "Sentry alerting level (info|warning|error|fatal|panic)")
	flag.DurationVar(&config.Log.Sentry.Timeout.Duration, "sentry-timeout", time.Second, "Sentry hook initialization timeout")

	// General
	flag.StringVar(&config.configFile, "config-file", "", "Path to a JSON file to read configuration from. Note: Will override options set earlier on the command line")
}

func (config *Config) loadConfigFromFile() error {
	if config.configFile == "" {
		return nil
	}
	jsonBlob, err := ioutil.ReadFile(config.configFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBlob, config)
}

func (config *Config) setLogLevel() error {
	level, err := log.ParseLevel(config.Log.Level)
	if err != nil {
		log.WithError(err).WithField("Level", config.Log.Level).Error("Bad level")
		return err
	}
	log.SetLevel(level)
	return nil
}

func (config *Config) setLogOutput() error {
	path := config.Log.File

	if len(path) == 0 {
		log.SetOutput(os.Stderr)
		return nil
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.WithError(err).Errorf("error opening file: %s", path)
		return err
	}

	log.SetOutput(f)
	return nil
}

func (config *Config) setLogFormat() {
	format := strings.ToUpper(config.Log.Format)
	if format == "JSON" {
		log.SetFormatter(&log.JSONFormatter{})
	} else if format == "TEXT" {
		log.SetFormatter(&log.TextFormatter{})
	} else {
		log.WithField("Format", format).Error("Unknown log format")
	}
}
