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
	Dc                     string
	Timeout                time.Interval
	RequestRetries         uint32
	AgentFailuresTolerance uint32
	ConsulNameSeparator    string
	IgnoredHealthChecks    string
	EnableTagOverride      bool
	LocalAgentHost         string
}

type Auth struct {
	Enabled  bool
	Username string
	Password string
}
