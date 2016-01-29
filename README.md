# marathon-consul [![Build Status](https://travis-ci.org/allegro/marathon-consul.svg?branch=master)](https://travis-ci.org/allegro/marathon-consul)[![Coverage Status](https://coveralls.io/repos/allegro/marathon-consul/badge.svg?branch=master&service=github)](https://coveralls.io/github/allegro/marathon-consul)


Register [Marathon](https://mesosphere.github.io/marathon/) Tasks as [Consul](https://www.consul.io/) Services for service discovery.

`marathon-consul` takes information provided by the [Marathon event bus](https://mesosphere.github.io/marathon/docs/event-bus.html) and
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

- Consul Agents should be available at every Mesos Slave, tasks will be registered at hosts their run on.
- Only tasks which are labeled as `consul` will be registered in Consul. By default the registered service name is equal to Marathon's application name. 
  A different name can be provided as the label's value, e.g. `consul:customName`. As an exception of the rule, for backward compatibility with the `0.3.x` branch, a value of `true` is resolved to the default name.
- Only services with tag specified by `consul-tag` property will be maintained. This tag is automatically added during registration. **Important**: it should be unique for every Marathon cluster connected to Consul.
- At least one HTTP healthcheck should be defined for a task. The task is registered when Marathon marks it's as alive.
- Provided HTTP healtcheck will be transferred to Consul.
- Labels with `tag` value will be converted to Consul tags, e.g. (note: `consul-tag` is set to `marathon`) `labels: ["public":"tag", "varnish":"tag", "env": "test"]` → `tags: ["public", "varnish", "marathon"]`.
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
consul-tag             | `marathon`            | Common tag name added to every service registered in Consul, should be unique for every Marathon-cluster connected to Consul
consul-timeout         | `1s`                  | Time limit for requests made by the Consul HTTP client. A Timeout of zero means no timeout
listen                 | `:4000`               | Accept connections at this address
log-level              | `info`                | Log level: panic, fatal, error, warn, info, or debug
log-format             | `text`                | Log format: JSON, text
log-file               |                       | Save logs to file (e.g.: `/var/log/marathon-consul.log`). If empty logs are published to STDERR
marathon-location      | `localhost:8080`      | Marathon URL
marathon-password      |                       | Marathon password for basic auth
marathon-protocol      | `http`                | Marathon protocol (http or https)
marathon-username      |                       | Marathon username for basic auth
metrics-interval       | `30s`                 | Metrics reporting [interval](https://golang.org/pkg/time/#Duration) **Note:** While using file configuration intervals should be provided in *nanoseconds*
metrics-location       |                       | Graphite URL (used when metrics-target is set to graphite)
metrics-prefix         | `default`             | Metrics prefix (resolved to `<hostname>.<app_name>` by default)
metrics-target         | `stdout`              | Metrics destination `stdout` or `graphite` (empty string disables metrics)
sync-enabled           | `true`                | Enable Marathon-consul scheduled sync
sync-interval          | `15m0s`               | Marathon-consul sync [interval](https://golang.org/pkg/time/#Duration) **Note:** While using file configuration intervals should be provided in *nanoseconds*
sync-leader            | `<hostname>:8080`     | Marathon cluster-wide node name (defaults to `<hostname>:8080`), the sync will run only if the node is the current [Marathon-leader](https://mesosphere.github.io/marathon/docs/rest-api.html#get-v2-leader)
sync-force             | `false`               | Force leadership-independent Marathon-consul sync (run always)


### Endpoints

Endpoint  | Description
----------|------------------------------------------------------------------------------------
`/health` | healthcheck - returns `OK`
`/events` | event sink - returns `OK` if all keys are set in an event, error message otherwise

### Known limitations

The following section describes known limitations with `marathon-consul`.

* Every marathon application needs to have a unique service name in Consul.
* In Marathon when a deployment changing the application's service name (by changing its `labels`) is being stopped, it changes app's configuration anyway.
  This means we loose the link between the app and the services registered with the old name in Consul. 
  Later on, if another deployment takes place, new services are registered with a new name, the old ones are not being deregistered though.
  A scheduled sync is required to wipe them out.

## Code

This project is based on

* [mesos-consul](https://github.com/CiscoCloud/mesos-consul)
* [marathon-consul](https://github.com/CiscoCloud/marathon-consul)

## License

Marathon-consul is released under the Apache 2.0 license (see [LICENSE](LICENSE))
