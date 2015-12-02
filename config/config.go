package config

import (
	"github.com/CiscoCloud/marathon-consul/consul"
	"github.com/CiscoCloud/marathon-consul/marathon"
	"github.com/CiscoCloud/marathon-consul/metrics"
	log "github.com/Sirupsen/logrus"
	flag "github.com/ogier/pflag"
	"time"
)

type Config struct {
	Consul consul.ConsulConfig
	Web    struct {
		Listen string
	}
	Marathon marathon.Config
	Metrics  metrics.Config
	LogLevel string
}

func New() (config *Config) {
	config = &Config{
		Marathon: marathon.Config{},
	}
	config.parseFlags()
	config.setLogLevel()

	return config
}

func (config *Config) parseFlags() {
	// Consul
	flag.BoolVar(&config.Consul.Enabled, "consul", true, "Use Consul backend")
	flag.StringVar(&config.Consul.Port, "consul-port", "8500", "Consul port")
	flag.BoolVar(&config.Consul.Auth.Enabled, "consul-auth", false, "Use Consul with authentication")
	flag.StringVar(&config.Consul.Auth.Username, "consul-auth-username", "", "The basic authentication username")
	flag.StringVar(&config.Consul.Auth.Password, "consul-auth-password", "", "The basic authentication password")
	flag.BoolVar(&config.Consul.SslEnabled, "consul-ssl", false, "Use HTTPS when talking to Consul")
	flag.BoolVar(&config.Consul.SslVerify, "consul-ssl-verify", true, "Verify certificates when connecting via SSL")
	flag.StringVar(&config.Consul.SslCert, "consul-ssl-cert", "", "Path to an SSL client certificate to use to authenticate to the Consul server")
	flag.StringVar(&config.Consul.SslCaCert, "consul-ssl-ca-cert", "", "Path to a CA certificate file, containing one or more CA certificates to use to validate the certificate sent by the Consul server to us")
	flag.StringVar(&config.Consul.Token, "consul-token", "", "The Consul ACL token")

	// Web
	flag.StringVar(&config.Web.Listen, "listen", ":4000", "accept connections at this address")

	// Marathon
	flag.StringVar(&config.Marathon.Location, "marathon-location", "localhost:8080", "marathon URL")
	flag.StringVar(&config.Marathon.Protocol, "marathon-protocol", "http", "marathon protocol (http or https)")
	flag.StringVar(&config.Marathon.Username, "marathon-username", "", "marathon username for basic auth")
	flag.StringVar(&config.Marathon.Password, "marathon-password", "", "marathon password for basic auth")

	// Metrics
	flag.StringVar(&config.Metrics.Target, "metrics-target", "stdout", "Metrics destination stdout or graphite")
	flag.StringVar(&config.Metrics.Prefix, "metrics-prefix", "default", "Metrics prefix (default is resolved to <hostname>.<app_name>")
	config.Metrics.Interval = (time.Duration)(*flag.Int64("metrics-interval", 30, "interval in seconds")) * time.Second
	flag.StringVar(&config.Metrics.Addr, "metrics-location", "", "Graphite URL (used when metrics-target is set to graphite)")

	// General
	flag.StringVar(&config.LogLevel, "log-level", "info", "log level: panic, fatal, error, warn, info, or debug")

	flag.Parse()
}

func (config *Config) setLogLevel() {
	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.WithField("level", config.LogLevel).Fatal("bad level")
	}
	log.SetLevel(level)
}
