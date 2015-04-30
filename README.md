# marathon-consul

Marathon to Consul bridge for metadata discovery.

`marathon-consul` takes information provided by the Marathon event bus and
forwards it to Consul's KV tree.

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc/generate-toc again -->
**Table of Contents**

- [marathon-consul](#marathon-consul)
    - [Arguments](#arguments)
    - [Endpoints](#endpoints)
    - [Keys and Values](#keys-and-values)

<!-- markdown-toc end -->

## Comparison to other metadata bridges

### `haproxy-marathon-bridge`

This project has similar goals (to enable metadata usage in templates.) However,
`haproxy-marathon-bridge` uses cron instead of the event bus, so it only updates
once per minute. It is also limited to haproxy, where `marathon-consul` in
conjuction with [consul-template](https://github.com/hashicorp/consul-template)
can update anything you can write a configuration file for.

## Arguments

Argument | Default | Description
---------|---------|------------
`listen` | :4000 | accept connections at this address
`registry` | http://localhost:8500 | root location of the Consul registry
`registry-auth` | None | basic auth for the Consul registry
`registry-datacenter` | None | datacenter to use in writes
`registry-token` | None | Consul registry ACL token
`registry-noverify` | False | don't verify registry SSL certificates
`verbose` | False | enable verbose logging

## Endpoints

Endpoint | Description
---------|------------
`/health` | healthcheck - returns `OK`
`/events` | event sink - returns `OK` if all keys are set in an event, error message otherwise

The Marathon event bus should point to `/events`. You can bootstrap the event
subscription like this (substituting the locations for your own, of course, but
this plays nicely with
[mesos-consul](https://github.com/ciscocloud/mesos-consul)):

    curl -X POST 'http://marathon.service.consul:8080/v2/eventSubscriptions?callbackUrl=http://marathon-consul.service.consul:4000/events'

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
