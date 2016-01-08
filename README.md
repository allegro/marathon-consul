# marathon-consul [![Build Status](https://travis-ci.org/allegro/marathon-consul.svg?branch=master)](https://travis-ci.org/allegro/marathon-consul)[![Coverage Status](https://coveralls.io/repos/allegro/marathon-consul/badge.svg?branch=master&service=github)](https://coveralls.io/github/allegro/marathon-consul)


Register Marathon Tasks as Consul Services for service discovery.

`marathon-consul` takes information provided by the Marathon event bus and
forwards it to Consul agents. It also re-syncs all the information from Marathon 
to Consul on startup and repeats it with given interval.

## Running the program

Just run `marathon-consul`.
You can also add some [options](#options).

The Marathon event bus should point to [`/events`](#endpoints). You can
set up the event subscription with a call similar to this one:

```
curl -X POST 'http://marathon.service.consul:8080/v2/eventSubscriptions?callbackUrl=http://marathon-consul.service.consul:4000/events'
```

## Building from source

To simply compile and run the source code:

```
go run main.go [options]
```

To build the binary:

```
make build
```

To run the tests:

```
make test
```


## Usage

- Consul Agents should be available at every Mesos Slave, tasks will be registered at host their run on.
- Only tasks which are labeled as `consul:true` will be registered in Consul.
- Only services with tag `marathon` will be maintained. This tag is automatically added during registration.
- At least one HTTP healthcheck should be defined for a task. The task is registered when Marathon marks it's as alive.
- Provided HTTP healtcheck will be transferred to Consul.
- Labels with `tag` value will be converted to Consul tags, `marathon` tag is added by default
 (e.g, `labels: ["public":"tag", "varnish":"tag", "env": "test"]` → `tags: ["public", "varnish", "marathon"]`).
- The scheduled Marathon-consul sync may run in two modes:
    - Only on node that is the current [Marathon-leader](https://mesosphere.github.io/marathon/docs/rest-api.html#get-v2-leader), `sync-leader` parameter should be set to `hostname:port` the current node appears in the Marathon cluster. 
      This mode is **enabled by default** and the `sync-leader` property is set to the hostname resolved by OS.
      Note that there is a difference between `sync-leader` and `marathon-location`: `sync-leader` is used for node leadership detection (should be set to cluster-wide node name), while `marathon-location` is used for connection purpose (may be set to `localhost`)
    - On every node, `sync-force` parameter should be set to `true`

### Options

Argument               | Default               | Description
-----------------------|-----------------------|------------------------------------------------------
config-file            |                       | Path to a JSON file to read configuration from. **Note:** Will override options set earlier on the command line. See [example](debian/config.json).
consul-auth            | `false`               | Use Consul with authentication
consul-auth-password   |                       | The basic authentication password
consul-auth-username   |                       | The basic authentication username
consul-port            | `8500`                | Consul port
consul-ssl             | `false`               | Use HTTPS when talking to Consul
consul-ssl-ca-cert     |                       | Path to a CA certificate file, containing one or more CA certificates to use to validate the certificate sent by the Consul server to us
consul-ssl-cert        |                       | Path to an SSL client certificate to use to authenticate to the Consul server
consul-ssl-verify      | `true`                | Verify certificates when connecting via SSL
consul-token           |                       | The Consul ACL token
listen                 | `:4000`               | Accept connections at this address
log-level              | `info`                | Log level: panic, fatal, error, warn, info, or debug
log-format             | `text`                | Log format: JSON, text
marathon-location      | `localhost:8080`      | Marathon URL
marathon-password      |                       | Marathon password for basic auth
marathon-protocol      | `http`                | Marathon protocol (http or https)
marathon-username      |                       | Marathon username for basic auth
metrics-interval       | `30s`                 | Metrics reporting [interval](https://golang.org/pkg/time/#Duration) **Note:** While using file configuration intervals should be provided in *nanoseconds*
metrics-location       |                       | Graphite URL (used when metrics-target is set to graphite)
metrics-prefix         | `default`             | Metrics prefix (resolved to `<hostname>.<app_name>` by default)
metrics-target         | `stdout`              | Metrics destination stdout or graphite
sync-enabled           | `true`                | Enable Marathon-consul scheduled sync
sync-interval          | `15m0s`               | Marathon-consul sync [interval](https://golang.org/pkg/time/#Duration) **Note:** While using file configuration intervals should be provided in *nanoseconds*
sync-leader            | `<hostname>:8080`     | Marathon cluster-wide node name (defaults to `<hostname>:8080`), the sync will run only if the node is the current [Marathon-leader](https://mesosphere.github.io/marathon/docs/rest-api.html#get-v2-leader)
sync-force             | `false`               | Force leadership-independent Marathon-consul sync (run always)


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

This project is based on

* [mesos-consul](https://github.com/CiscoCloud/mesos-consul)
* [marathon-consul](https://github.com/CiscoCloud/marathon-consul)

## License

Marathon-consul is released under the Apache 2.0 license (see [LICENSE](LICENSE))
