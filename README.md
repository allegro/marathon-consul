
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

Argument               | Default               | Description
-----------------------|-----------------------|----------------------------------------
`listen`               | :4000                 | accept connections at this address
`registry`             | http://localhost:8500 | root location of the Consul registry
`registry-auth`        | None                  | basic auth for the Consul registry
`registry-datacenter`  | None                  | datacenter to use in writes
`registry-token`       | None                  | Consul registry ACL token
`registry-noverify`    | False                 | don't verify registry SSL certificates
`verbose`              | False                 | enable verbose logging

### Adding New Root Certificate Authorities

If you're running Consul behind an SSL proxy like Nginx, you're probably going
to want to add the CA for your certificate to the trusted store in the container
so you can avoid using `--registry-noverify`. For that purpose, any certificates
added in a volume at `/usr/local/share/ca-certificates/` will be added to the
root certificates in the container on boot.

### Endpoints

Endpoint  | Description
----------|------------------------------------------------------------------------------------
`/health` | healthcheck - returns `OK`
`/events` | event sink - returns `OK` if all keys are set in an event, error message otherwise

## Keys and Values

The entire app configuration is forwarded to Consul as a JSON blob. It might
looks something like this (example from the Marathon documentation):

```
{
    "id": "/product/service/my-app",
    "cmd": "env && sleep 300",
    "args": ["/bin/sh", "-c", "env && sleep 300"],
    "container": {
        "type": "DOCKER",
        "docker": {
            "image": "group/image",
            "network": "BRIDGE",
            "portMappings": [
                {
                    "containerPort": 8080,
                    "hostPort": 0,
                    "servicePort": 9000,
                    "protocol": "tcp"
                },
                {
                    "containerPort": 161,
                    "hostPort": 0,
                    "protocol": "udp"
                }
            ],
            "privileged": false,
            "parameters": [
                { "key": "a-docker-option", "value": "xxx" },
                { "key": "b-docker-option", "value": "yyy" }
            ]
        },
        "volumes": [
            {
                "containerPath": "/etc/a",
                "hostPath": "/var/data/a",
                "mode": "RO"
            },
            {
                "containerPath": "/etc/b",
                "hostPath": "/var/data/b",
                "mode": "RW"
            }
        ]
    },
    "cpus": 1.5,
    "mem": 256.0,
    "env": {
        "LD_LIBRARY_PATH": "/usr/local/lib/myLib"
    },
    "executor": "",
    "constraints": [
        ["attribute", "OPERATOR", "value"]
    ],
    "labels": {
        "environment": "staging"
    },
    "healthChecks": [
        {
            "protocol": "HTTP",
            "path": "/health",
            "gracePeriodSeconds": 3,
            "intervalSeconds": 10,
            "portIndex": 0,
            "timeoutSeconds": 10,
            "maxConsecutiveFailures": 3
        },
        {
            "protocol": "TCP",
            "gracePeriodSeconds": 3,
            "intervalSeconds": 5,
            "portIndex": 1,
            "timeoutSeconds": 5,
            "maxConsecutiveFailures": 3
        },
        {
            "protocol": "COMMAND",
            "command": { "value": "curl -f -X GET http://$HOST:$PORT0/health" },
            "maxConsecutiveFailures": 3
        }
    ],
    "instances": 3,
    "ports": [
        8080,
        9000
    ],
    "backoffSeconds": 1,
    "backoffFactor": 1.15,
    "uris": [
        "https://raw.github.com/mesosphere/marathon/master/README.md"
    ],
    "dependencies": ["/product/db/mongo", "/product/db", "../../db"],
    "upgradeStrategy": {
        "minimumHealthCapacity": 0.5,
        "maximumOverCapacity": 0.2
    },
    "version": "2014-03-01T23:29:30.158Z"
}
```

## License

marathon-consul is released under the Apache 2.0 license (see [LICENSE](LICENSE))
