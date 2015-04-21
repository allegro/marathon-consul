# Marathon Forwarder

Marathon Forwarder forwards app metadata to a Consul KV store

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc/generate-toc again -->
**Table of Contents**

- [Marathon Forwarder](#marathon-forwarder)
    - [Arguments](#arguments)
    - [Endpoints](#endpoints)
    - [Keys and Values](#keys-and-values)

<!-- markdown-toc end -->

## Arguments

Argument | Default | Description
---------|---------|------------
`consul` | "http://localhost:8500" | Consul location
`datacenter` | blank | Consul datacenter
`noverify` | false | don't verify certificates when connecting to Consul
`parallelism` | 4 | set this many keys at once (per request)
`serve` | ":8000" | accept connections at this address
`token` | blank | consul ACL token
`user` and `pass` | blank | if both are set, use basic auth to connect to Consul
`verbose` | false | print debug information for every request

## Endpoints

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
