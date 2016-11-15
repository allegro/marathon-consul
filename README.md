# marathon-consul [![Build Status](https://travis-ci.org/allegro/marathon-consul.svg?branch=master)](https://travis-ci.org/allegro/marathon-consul)&nbsp;[![Coverage Status](https://coveralls.io/repos/allegro/marathon-consul/badge.svg?branch=master&service=github)](https://coveralls.io/github/allegro/marathon-consul)&nbsp;[![Go Report Card](https://goreportcard.com/badge/github.com/allegro/marathon-consul)](https://goreportcard.com/report/github.com/allegro/marathon-consul)&nbsp;[![Download latest version](https://api.bintray.com/packages/allegro/deb/marathon-consul/images/download.svg)](https://bintray.com/allegro/deb/marathon-consul/_latestVersion)


Register [Marathon](https://mesosphere.github.io/marathon/) Tasks as [Consul](https://www.consul.io/) Services for service discovery.

`marathon-consul` takes information provided by the [Marathon event bus](https://mesosphere.github.io/marathon/docs/event-bus.html) and
forwards it to Consul agents. It also re-syncs all the information from Marathon
to Consul on startup and repeats it with given interval.

## Code

This project is based on

* [mesos-consul](https://github.com/CiscoCloud/mesos-consul)
* [marathon-consul](https://github.com/CiscoCloud/marathon-consul)

### Differences

* CiscoCloud/marathon-consul copies application information to Consul KV while
allegro/marathon-consul registers tasks as Consul services
(it is more similar to CiscoCloud/mesos-consul)
* CiscoCloud/mesos-consul uses polling while allegro/marathon-consul uses
[Marathon's event bus](https://mesosphere.github.io/marathon/docs/event-bus.html)
to detect changes
* CiscoCloud/marathon-consul is no longer developed
(see [comment](https://github.com/CiscoCloud/marathon-consul/issues/17#issuecomment-161678453))


## Installation

### Installing from source code

To simply compile and run the source code:

```
go run main.go [options]
```

To run the tests:

```
make test
```

To build the binary:

```
make build
```

### Installing from binary distribution

Binary distribution of `marathon-consul` can be downloaded directly from [the releases page](https://github.com/allegro/marathon-consul/releases).
Download the build dedicated to your OS. After unpacking the archive, run `marathon-consul` binary. You can also add some [options](#options), for example:

```bash
marathon-consul --marathon-location=marathon.service.consul:8080 --sync-interval=5m --log-level=debug
```

### Installing via APT package manager

If you are a Debian/Ubuntu user, you can easily install `marathon-consul` as a `deb` package using `APT` package manager. Both `upstart` and `systemd` service managers are supported.
All releases are published as `deb` packages to [our repository at Bintray](https://bintray.com/allegro/deb/marathon-consul/view).

To install `marathon-consul` with `apt-get`, simply follow the instructions:

```bash
# add our public key to apt
curl -s https://bintray.com/user/downloadSubjectPublicKey?username=allegro | sudo apt-key add -
# add the repository url
echo "deb http://dl.bintray.com/v1/content/allegro/deb /" | sudo tee /etc/apt/sources.list.d/marathon-consul.list
# update apt cache
sudo apt-get -y update
# install latest release of marathon-consul
sudo apt-get -qy install marathon-consul
```

Run it with `service marathon-consul start`. The configuration file is located at `/etc/marathon-consul.d/config.json`.

### Installing with Docker

To build docker image run
```bash
make docker
```
Then you can run it with
```bash
docker run -d -P allegro/marathon-consul [options]
```
If you want to use SSL you will need to expose cert store to Docker. The Following line is only example,
your cert store might be different depending on your system.
```bash
docker run '/etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt' -P  allegro/marathon-consul
```

## Setting up `marathon-consul` after installation

The Marathon [event bus](https://mesosphere.github.io/marathon/docs/event-bus.html) should point to [`/events`](#endpoints). You can set up the event subscription with a call similar to this one:
```
curl -X POST 'http://marathon.service.consul:8080/v2/eventSubscriptions?callbackUrl=http://marathon-consul.service.consul:4000/events'
```
The event subscription should be set to `localhost` to reduce network traffic.


## Usage

### Marathon masters

- marathon-consul should be installed on all Marathon masters

### Mesos slaves

- Consul Agents should be available on every Mesos slave.
- Tasks will be registered at the Mesos slave they run on.

### Tagging tasks

- Task labels are used by marathon-consul to register tasks in Consul.
- Only tasks which are labeled as `consul` will be registered in Consul.
  If the `consul` label is left blank like `"consul": ""`, the task will be registered with the app name
by default. A different name can be provided as the label's value, e.g.
`"consul": "customName"`. As an exception to this rule, for backward compatibility with the `0.3.x` branch, a value of `true` is resolved to the default name.
```
  "id": "my-new-app",
  "labels": {
    "consul": ""
  }
```
- Only services with tag specified by `consul-tag` property will be maintained. By default, `"consul-tag": "marathon"` is automatically added when the task is registered.
- Labels with a *value* of `tag` are converted to consul-tags, and appear in Consul as ServiceTags.
- For example, we can set these tags in an app definition like:
```
  "id": "my-new-app",
  "labels": {
    "consul": "",
    "varnish": "tag",
    "metrics": "tag"
  }
```
- If marathon-consul registers the app with Consul, we can then query Consul and see these tags appear:
```
curl -X GET http://localhost:8500/v1/catalog/service/my-new-app
...
"ServiceName": "my-new-app",
"ServiceTags": [
  "marathon",
  "varnish",
  "metrics",
  "marathon-task:my-new-app.6a95bb03-6ad3-11e6-beaf-080027a7aca0"
],

```
- Every service registration contains an additional tag `marathon-task` specifying the Marathon task id related to this registration.
- If there are multiple ports in use for the same app, note that only the first one will be registered by marathon-consul in Consul.

If you need to register your task under multiple ports, refer to *Advanced usage* section below.

### Task healthchecks

- At least one HTTP healthcheck should be defined for a task. The task is registered when Marathon marks it as alive.
- The provided HTTP healthcheck will be transferred to Consul.
- See [this](https://mesosphere.github.io/marathon/docs/health-checks.html)
for more details.

### Sync

- The scheduled Marathon-consul sync may run in two modes:
    - Only on node that is the current [Marathon-leader](https://mesosphere.github.io/marathon/docs/rest-api.html#get-v2-leader), `sync-leader` parameter should be set to `hostname:port` the current node appears in the Marathon cluster. 
      This mode is **enabled by default** and the `sync-leader` property is set to the hostname resolved by OS.
      Note that there is a difference between `sync-leader` and `marathon-location`: `sync-leader` is used for node leadership detection (should be set to cluster-wide node name), while `marathon-location` is used for connection purpose (may be set to `localhost`)
    - On every node, `sync-force` parameter should be set to `true`

### Options

Argument                    | Default         | Description
----------------------------|-----------------|------------------------------------------------------
config-file                 |                 | Path to a JSON file to read configuration from. Note: Will override options set earlier on the command line
consul-auth                 | `false`         | Use Consul with authentication
consul-auth-password        |                 | The basic authentication password
consul-auth-username        |                 | The basic authentication username
consul-ignored-healthchecks |                 | A comma separated blacklist of Marathon health check types that will not be migrated to Consul, e.g. command,tcp
consul-name-separator       | `.`             | Separator used to create default service name for Consul
consul-get-services-retry   | `3`             | Number of retries on failure when performing requests to Consul. Each retry uses different cached agent
consul-max-agent-failures   | `3`             | Max number of consecutive request failures for agent before removal from cache
consul-port                 | `8500`          | Consul port
consul-ssl                  | `false`         | Use HTTPS when talking to Consul
consul-ssl-ca-cert          |                 | Path to a CA certificate file, containing one or more CA certificates to use to validate the certificate sent by the Consul server to us
consul-ssl-cert             |                 | Path to an SSL client certificate to use to authenticate to the Consul server
consul-ssl-verify           | `true`          | Verify certificates when connecting via SSL
consul-tag                  | `marathon`      | Common tag name added to every service registered in Consul, should be unique for every Marathon-cluster connected to Consul
consul-timeout              | `3s`            | Time limit for requests made by the Consul HTTP client. A Timeout of zero means no timeout
consul-token                |                 | The Consul ACL token
events-queue-size           | `1000`          | Size of events queue
event-max-size              | `4096`          | Maximum size of event to process (bytes)
listen                      | `:4000`         | Accept connections at this address
log-file                    |                 | Save logs to file (e.g.: `/var/log/marathon-consul.log`). If empty logs are published to STDERR
log-format                  | `text`          |  Log format: JSON, text
log-level                   | `info`          | Log level: panic, fatal, error, warn, info, or debug
marathon-location           | `localhost:8080`| Marathon URL
marathon-password           |                 | Marathon password for basic auth
marathon-protocol           | `http`          | Marathon protocol (http or https)
marathon-ssl-verify         | `true`          | Verify certificates when connecting via SSL
marathon-timeout            | `30s`           | Time limit for requests made by the Marathon HTTP client. A Timeout of zero means no timeout
marathon-username           |                 | Marathon username for basic auth
metrics-interval            | `30s`           | Metrics reporting interval
metrics-location            |                 | Graphite URL (used when metrics-target is set to graphite)
metrics-prefix              | `default`       | Metrics prefix (default is resolved to <hostname>.<app_name>
metrics-target              | `stdout`        | Metrics destination stdout or graphite (empty string disables metrics)
sync-enabled                | `true`          | Enable Marathon-consul scheduled sync
sync-force                  | `false`         | Force leadership-independent Marathon-consul sync (run always)
sync-interval               | `15m0s`         | Marathon-consul sync interval
sync-leader                 |                 | Marathon cluster-wide node name (defaults to <hostname>:8080), the sync will run only if the specified node is the current Marathon-leader
workers-pool-size           | `10`            | Number of concurrent workers processing events

### Endpoints

Endpoint  | Description
----------|------------------------------------------------------------------------------------
`/health` | healthcheck - returns `OK`
`/events` | event sink - returns `OK` if all keys are set in an event, error message otherwise

## Advanced usage

### Register under multiple ports

If you need to map your Marathon task into multiple service registrations in Consul, you can configure marathon-consul 
via Marathon's `portDefinitions`:

```
  "id": "my-new-app",
  "labels": {
    "consul": "",
    "common-tag": "tag"
  },
  "portDefinitions": [
    {
      "labels": {
        "consul": "my-app-custom-name"
      }
    },
    {
      "labels": {
        "consul": "my-app-other-name",
        "specific-tag": "tag"
      }
    }
  ]
```

This configuration would result in two service registrations:

```
curl -X GET http://localhost:8500/v1/catalog/service/my-app-custom-name
...
"ServiceName": "my-app-custom-name",
"ServiceTags": [
  "marathon",
  "common-tag",
  "marathon-task:my-new-app.6a95bb03-6ad3-11e6-beaf-080027a7aca0"
],
"ServicePort": 31292,
...

curl -X GET http://localhost:8500/v1/catalog/service/my-app-other-name
...
"ServiceName": "my-app-other-name",
"ServiceTags": [
  "marathon",
  "common-tag",
  "specific-tag",
  "marathon-task:my-new-app.6a95bb03-6ad3-11e6-beaf-080027a7aca0"
],
"ServicePort": 31293,
...
``` 

If any port definition contains the `consul` label, then advanced configuration mode is enabled. As a result, only the ports 
containing this label are registered, under the name specified as the label's value – with empty value resolved to default name.
Names don't have to be unique – you can have multiple registrations under the same name, but on different ports, 
perhaps with different tags. Note that the `consul` label still needs to be present in the top-level application labels, even
though its value won't have any effect.

Tags configured in the top-level application labels will be added to all registrations. Tags configured in the port definition 
labels will be added only to corresponding registrations.

All registrations share the same `marathon-task` tag.

## Migration to version 1.x.x

Until 1.x.x marathon-consul would register services in Consul with registration id equal to related Marathon task id. Since 1.x.x registration ids are different and
an additional tag, `marathon-task`, is added to each registration.

If you update marathon-consul from version 0.x.x to 1.x.x, expect the synchronization phase during the first startup to 
reregister all healthy services managed by marathon-consul to the new format. Unhealthy services will get deregistered in the process.

## Known limitations

The following section describes known limitations in `marathon-consul`.

* In Marathon when a deployment changing the application's service name (by changing its `labels`) is being stopped, it changes app's configuration anyway.
  This means we loose the link between the app and the services registered with the old name in Consul.
  Later on, if another deployment takes place, new services are registered with a new name, the old ones are not being deregistered though.
  A scheduled sync is required to wipe them out.

## License

Marathon-consul is released under the Apache 2.0 license (see [LICENSE](LICENSE))
