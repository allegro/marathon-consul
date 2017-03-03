package consul

import "github.com/allegro/marathon-consul/time"

type Config struct {
	Auth                   Auth
	Port                   string
	SslEnabled             bool
	SslVerify              bool
	SslCert                string
	SslCaCert              string
	Token                  string
	Tag                    string
	Timeout                time.Interval
	RequestRetries         uint64
	AgentFailuresTolerance uint64
	ConsulNameSeparator    string
	IgnoredHealthChecks    string
}

type Auth struct {
	Enabled  bool
	Username string
	Password string
}
