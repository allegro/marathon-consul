package consul

import "time"

type Config struct {
	Auth                   Auth
	Port                   string
	SslEnabled             bool
	SslVerify              bool
	SslCert                string
	SslCaCert              string
	Token                  string
	Tag                    string
	Timeout                time.Duration
	RequestRetries         uint32
	AgentFailuresTolerance uint32
	ConsulNameSeparator    string
}

type Auth struct {
	Enabled  bool
	Username string
	Password string
}
