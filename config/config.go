package config

import (
	"crypto/tls"
	"errors"
	"github.com/hashicorp/consul/api"
	flag "github.com/ogier/pflag"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrBadCredentials = errors.New("credentials must be of the form `user:pass`")
)

type Config struct {
	Registry Registry
	Web      struct {
		Listen string
	}
	Verbose bool
}

func New() (config *Config) {
	config = &Config{
		Registry: Registry{},
		Verbose:  false,
	}
	config.parseFlags()

	return config
}

func (config *Config) parseFlags() {
	// registry
	flag.StringVar(&config.Registry.Auth, "registry-auth", "", "Registry basic auth")
	flag.StringVar(&config.Registry.Datacenter, "registry-datacenter", "", "Registry datacenter")
	flag.StringVar(&config.Registry.Location, "registry", "http://localhost:8500", "Registry location")
	flag.StringVar(&config.Registry.Token, "registry-token", "", "Registry ACL token")
	flag.BoolVar(&config.Registry.NoVerifySSL, "registry-noverify", false, "don't verify registry SSL certificates")
	flag.StringVar(&config.Registry.Prefix, "registry-prefix", "marathon", "prefix for all values sent to the registry")

	// Web
	flag.StringVar(&config.Web.Listen, "listen", ":4000", "accept connections at this address")

	// General
	flag.BoolVar(&config.Verbose, "verbose", false, "enable verbose logging")

	flag.Parse()
}

type Registry struct {
	Auth        string
	Datacenter  string
	Location    string
	Token       string
	NoVerifySSL bool
	Prefix      string
}

func (r Registry) GetAuth() (auth *api.HttpBasicAuth, err error) {
	if r.Auth == "" {
		return nil, nil
	}

	creds := strings.SplitN(r.Auth, ":", 2)
	if len(creds) != 2 {
		return nil, ErrBadCredentials
	}

	auth = &api.HttpBasicAuth{
		Username: creds[0],
		Password: creds[1],
	}

	return auth, err
}

func (r Registry) Config() (*api.Config, error) {
	url, err := url.Parse(r.Location)
	if err != nil {
		return nil, err
	}

	auth, err := r.GetAuth()
	if err != nil {
		return nil, err
	}

	config := &api.Config{
		Address:    url.Host,
		Scheme:     url.Scheme,
		Datacenter: r.Datacenter,
		HttpAuth:   auth,
		Token:      r.Token,
		HttpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: r.NoVerifySSL,
				},
			},
		},
	}

	return config, nil
}
