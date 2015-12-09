package consul

type ConsulConfig struct {
	Auth       Auth
	Port       string
	SslEnabled bool
	SslVerify  bool
	SslCert    string
	SslCaCert  string
	Token      string
}

type Auth struct {
	Enabled  bool
	Username string
	Password string
}
