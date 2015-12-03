# marathon-consul


Register Marathon Tasks as Consul Services for service discovery.

`marathon-consul` takes information provided by the Marathon event bus and
forwards it to Consul's services. It also re-syncs all the information from
Marathon to Consul on startup and repeat it in given interval.

## Running

Just run `marathon-consul`.
You can also add some [options](#options).

The Marathon event bus should point to [`/events`](#endpoints). You can
set up the event subscription with a call similar to this one:

```
curl -X POST 'http://marathon.service.consul:8080/v2/eventSubscriptions?callbackUrl=http://marathon-consul.service.consul:4000/events'
```

## Usage

- Only tasks which are labeled as `consul:true` will be registered in Consul.
- Only services with tag `marathon` will be maintained. This tag is automatically added when new instance is registered.
- Task is registered when Marathon marks it's as alive. So tasks must have defined healthchecks
- HTTP healtcheck will be transfered to Consul
- Labels with `tag` value will be converted to Consul tags, `marathon` tag is added by default
 (e.g, `labels: ["public":"tag", "varnish":"tag", "env": "test"]` → `tags: ["public", "varnish", "marathon"]`)   

### Options

Argument               | Default               | Description
-----------------------|-----------------------|------------------------------------------------------
consul                 | `true`                | Use Consul backend
consul-auth            | `false`               | Use Consul with authentication
consul-auth-password   |                       | The basic authentication password
consul-auth-username   |                       | The basic authentication username
consul-port            | `8500`                | Consul port
consul-ssl             | `false`               | Use HTTPS when talking to Consul
consul-ssl-ca-cert     |                       | Path to a CA certificate file, containing one or more CA certificates to use to validate the certificate sent by the Consul server to us
consul-ssl-cert        |                       | Path to an SSL client certificate to use to authenticate to the Consul server
consul-ssl-verify      | `true`                | Verify certificates when connecting via SSL
consul-token           |                       | The Consul ACL token
listen                 | :4000                 | Accept connections at this address
log-level              | info                  | Log level: panic, fatal, error, warn, info, or debug
marathon-location      | localhost:8080        | Marathon URL
marathon-password      |                       | Marathon password for basic auth
marathon-protocol      | http                  | Marathon protocol (http or https)
marathon-username      |                       | Marathon username for basic auth
metrics-interval       | 30s                   | Metrics reporting [interval](https://golang.org/pkg/time/#Duration)
metrics-location       |                       | Graphite URL (used when metrics-target is set to graphite)
metrics-prefix         | default               | Metrics prefix (default is resolved to <hostname>.<app_name>
metrics-target         | stdout                | Metrics destination stdout or graphite
sync-interval          | 15m0s                 | Marathon-consul sync interval


### Adding New Root Certificate Authorities

If you're running Consul behind an SSL proxy like Nginx, you're probably going
to want to add the CA for your certificate to the trusted store in the container
so you can avoid using `--consul-ssl-verify`. For that purpose, any certificates
added in a volume at `/usr/local/share/ca-certificates/` will be added to the
root certificates in the container on boot.

### Endpoints

Endpoint  | Description
----------|------------------------------------------------------------------------------------
`/health` | healthcheck - returns `OK`
`/events` | event sink - returns `OK` if all keys are set in an event, error message otherwise

## Code

Project is based on

* [mesos-consul](https://github.com/CiscoCloud/mesos-consul)
* [marathon-consul](https://github.com/CiscoCloud/marathon-consul)

## License

marathon-consul is released under the Apache 2.0 license (see [LICENSE](LICENSE))
