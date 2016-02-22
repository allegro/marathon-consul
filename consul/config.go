package consul

import "time"

type ConsulConfig struct {
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
}

type Auth struct {
	Enabled  bool
	Username string
	Password string
}
