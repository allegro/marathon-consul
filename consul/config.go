package consul

import "time"

type ConsulConfig struct {
	Auth       Auth
	Port       string
	SslEnabled bool
	SslVerify  bool
	SslCert    string
	SslCaCert  string
	Token      string
	Tag        string
	Timeout    time.Duration
}

type Auth struct {
	Enabled  bool
	Username string
	Password string
}
