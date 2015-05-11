
# marathon-consul

Marathon to Consul bridge for metadata discovery.

`marathon-consul` takes information provided by the Marathon event bus and
forwards it to Consul's KV tree.

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc/generate-toc again -->
**Table of Contents**

- [marathon-consul](#marathon-consul)
    - [Comparison to other metadata bridges](#comparison-to-other-metadata-bridges)
        - [`haproxy-marathon-bridge`](#haproxy-marathon-bridge)
    - [Building](#building)
    - [Running](#running)
    - [Usage](#usage)
        - [Options](#options)
        - [Adding New Root Certificate Authorities](#adding-new-root-certificate-authorities)
        - [Endpoints](#endpoints)
    - [Keys and Values](#keys-and-values)
    - [License](#license)

<!-- markdown-toc end -->

## Comparison to other metadata bridges

### `haproxy-marathon-bridge`

This project has similar goals (to enable metadata usage in templates.) However,
`haproxy-marathon-bridge` uses cron instead of the event bus, so it only updates
once per minute. It is also limited to haproxy, where `marathon-consul` in
conjunction with [consul-template](https://github.com/hashicorp/consul-template)
can update anything you can write a configuration file for.

## Building

```
docker build -t marathon-consul .
```

## Running

marathon-consul can be run in a Docker container via Marathon. If your Marathon
service is registered in consul, you can use `.service.consul` to find them,
otherwise change the vaules for your environment:

```
curl -X POST -d @marathon-consul.json -H "Content-Type: application/json" http://marathon.service.consul:8080/v2/apps'
```

Where `marathon-consul.json` is similar to (replacing the image with your image):

```
{
  "id": "marathon-consul",
  "args": ["--registry=https://consul.service.consul:8500"],
  "container": {
    "type": "DOCKER",
    "docker": {
      "image": "{{ marathon_consul_image }}:{{ marathon_consul_image_tag }}",
      "network": "BRIDGE",
      "portMappings": [{"containerPort": 4000, "hostPort": 4000, "protocol": "tcp"}]
    }
  },
  "constraints": [["hostname", "UNIQUE"]],
  "ports": [4000],
  "healthChecks": [{
    "protocol": "HTTP",
    "path": "/health",
    "portIndex": 0
  }],
  "instances": 1,
  "cpus": 0.1,
  "mem": 128
}
```

You can also add [options to authenticate against Consul](#options).

The Marathon event bus should point to [`/events``](#endpoints). You can
set up the event subscription with a call similar to this one:

```
curl -X POST 'http://marathon.service.consul:8080/v2/eventSubscriptions?callbackUrl=http://marathon-consul.service.consul:4000/events'
```

## Usage

### Options

Argument | Default | Description
---------|---------|------------
`listen` | :4000 | accept connections at this address
`registry` | http://localhost:8500 | root location of the Consul registry
`registry-auth` | None | basic auth for the Consul registry
`registry-datacenter` | None | datacenter to use in writes
`registry-token` | None | Consul registry ACL token
`registry-noverify` | False | don't verify registry SSL certificates
`verbose` | False | enable verbose logging

### Adding New Root Certificate Authorities

If you're running Consul behind an SSL proxy like Nginx, you're probably going
to want to add the CA for your certificate to the trusted store in the container
so you can avoid using `--registry-noverify`. For that purpose, any certificates
added in a volume at `/usr/local/share/ca-certificates/` will be added to the
root certificates in the container on boot.

### Endpoints

Endpoint | Description
---------|------------
`/health` | healthcheck - returns `OK`
`/events` | event sink - returns `OK` if all keys are set in an event, error message otherwise

## Keys and Values

The entire app configuration is forwarded to Consul. Simple values (strings,
ints, and so on) are represented by their string form. Complex values (lists,
hashes) are represented by their JSONified form.

Key | Example Value
----|--------------
`marathon/myApp/args` | `["arg"]`
`marathon/myApp/backoffFactor` | `0.5`
`marathon/myApp/backoffSeconds` | `1`
`marathon/myApp/cmd` | `command`
`marathon/myApp/constraints` | `[["HOSTNAME","unique"]]`
`marathon/myApp/container/docker/forcePullImage` | `true`
`marathon/myApp/container/docker/image` | `alpine`
`marathon/myApp/container/docker/network` | `BRIDGED`
`marathon/myApp/container/docker/parameters` | `[{"key":"hostname","value":"container.example.com"}]`
`marathon/myApp/container/docker/portMappings` | `[{"containerPort":8080,"hostPort":8080,"servicePort":0,"protocol":"tcp"}]`
`marathon/myApp/container/docker/privileged` | `true`
`marathon/myApp/container/type` | `DOCKER`
`marathon/myApp/container/volumes` | `[{"containerPath":"/tmp","hostPath":"/tmp/container","mode":"rw"}]`
`marathon/myApp/cpus` | `0.1`
`marathon/myApp/dependencies` | `["/otherApp"]`
`marathon/myApp/disk` | `128`
`marathon/myApp/env` | `{"HOME":"/tmp"}`
`marathon/myApp/executor` | `executor`
`marathon/myApp/healthChecks` | `[{"path":"/","portIndex":0,"protocol":"http","gracePeriodSeconds":30,"intervalSeconds":15,"timeoutSeconds":30,"maxConsecutiveFailures":5}]`
`marathon/myApp/id` | `/test`
`marathon/myApp/instances` | `2`
`marathon/myApp/labels` | `{"BALANCE":"yes"}`
`marathon/myApp/mem` | `256`
`marathon/myApp/ports` | `[10001]`
`marathon/myApp/requirePorts` | `true`
`marathon/myApp/storeUrls` | `["http://example.com/resource/"]`
`marathon/myApp/upgradeStrategy/maximumOverCapacity` | `1`
`marathon/myApp/upgradeStrategy/minimumHealthCapacity` | `1`
`marathon/myApp/uris` | `["http://example.com/"]`
`marathon/myApp/user` | `user`
`marathon/myApp/version` | `2015-01-01T00:00:00Z`

## License

marathon-consul is released under the Apache 2.0 license (see [LICENSE](LICENSE))
