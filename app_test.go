package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testApp = App{
	Args:           []string{"arg"},
	BackoffFactor:  0.5,
	BackoffSeconds: 1,
	Cmd:            "command",
	Constraints:    []string{"constraint"},
	Container: &Container{
		Type:    "DOCKER",
		Volumes: []Volume{Volume{ContainerPath: "/tmp", HostPath: "/tmp/container", Mode: "rw"}},
		Docker: &Docker{
			Image:          "alpine",
			Parameters:     []Parameter{Parameter{"hostname", "container.example.com"}},
			Privileged:     true,
			PortMappings:   []PortMapping{PortMapping{8080, 8080, 0, "tcp"}},
			Network:        "BRIDGED",
			ForcePullImage: true,
		},
	},
	CPUs:         0.1,
	Dependencies: []string{"/otherApp"},
	Disk:         128,
	Env:          map[string]string{"HOME": "/tmp"},
	Executor:     "executor",
	Labels:       map[string]string{"BALANCE": "yes"},
	HealthChecks: []HealthCheck{HealthCheck{
		Path:                   "/",
		PortIndex:              0,
		Protocol:               "http",
		GracePeriodSeconds:     30,
		IntervalSeconds:        15,
		TimeoutSeconds:         30,
		MaxConsecutiveFailures: 5,
	}},
	ID:           "/test",
	Instances:    2,
	Mem:          256,
	Ports:        []int{10001},
	RequirePorts: true,
	StoreUrls:    []string{"http://example.com/resource/"},
	UpgradeStrategy: UpgradeStrategy{
		MinimumHealthCapacity: 1.0,
		MaximumOverCapacity:   1.0,
	},
	Uris:    []string{"http://example.com/"},
	User:    "user",
	Version: "2015-01-01T00:00:00Z",
}

func TestKVs(t *testing.T) {
	t.Parallel()

	// making assertions on a map will be a little easier...
	kvs := map[string]string{}
	for _, kv := range testApp.KVs() {
		kvs[kv.Key] = string(kv.Value)
	}

	assert.Equal(t, `["arg"]`, kvs["marathon/test/args"])
	assert.Equal(t, "0.5", kvs["marathon/test/backoffFactor"])
	assert.Equal(t, "1", kvs["marathon/test/backoffSeconds"])
	assert.Equal(t, "command", kvs["marathon/test/cmd"])
	assert.Equal(t, `["constraint"]`, kvs["marathon/test/constraints"])
	assert.Equal(t, "DOCKER", kvs["marathon/test/container/type"])
	assert.Equal(t, `[{"containerPath":"/tmp","hostPath":"/tmp/container","mode":"rw"}]`, kvs["marathon/test/container/volumes"])
	assert.Equal(t, "alpine", kvs["marathon/test/container/docker/image"])
	assert.Equal(t, `[{"key":"hostname","value":"container.example.com"}]`, kvs["marathon/test/container/docker/parameters"])
	assert.Equal(t, "true", kvs["marathon/test/container/docker/privileged"])
	assert.Equal(t, `[{"containerPort":8080,"hostPort":8080,"servicePort":0,"protocol":"tcp"}]`, kvs["marathon/test/container/docker/portMappings"])
	assert.Equal(t, "BRIDGED", kvs["marathon/test/container/docker/network"])
	assert.Equal(t, "true", kvs["marathon/test/container/docker/forcePullImage"])
	assert.Equal(t, "0.1", kvs["marathon/test/cpus"])
	assert.Equal(t, `["/otherApp"]`, kvs["marathon/test/dependencies"])
	assert.Equal(t, "128", kvs["marathon/test/disk"])
	assert.Equal(t, `{"HOME":"/tmp"}`, kvs["marathon/test/env"])
	assert.Equal(t, "executor", kvs["marathon/test/executor"])
	assert.Equal(t, `{"BALANCE":"yes"}`, kvs["marathon/test/labels"])
	assert.Equal(t, `[{"path":"/","portIndex":0,"protocol":"http","gracePeriodSeconds":30,"intervalSeconds":15,"timeoutSeconds":30,"maxConsecutiveFailures":5}]`, kvs["marathon/test/healthChecks"])
	assert.Equal(t, "/test", kvs["marathon/test/id"])
	assert.Equal(t, "2", kvs["marathon/test/instances"])
	assert.Equal(t, "256", kvs["marathon/test/mem"])
	assert.Equal(t, `[10001]`, kvs["marathon/test/ports"])
	assert.Equal(t, "true", kvs["marathon/test/requirePorts"])
	assert.Equal(t, `["http://example.com/resource/"]`, kvs["marathon/test/storeUrls"])
	assert.Equal(t, "1", kvs["marathon/test/upgradeStrategy/minimumHealthCapacity"])
	assert.Equal(t, "1", kvs["marathon/test/upgradeStrategy/maximumOverCapacity"])
	assert.Equal(t, `["http://example.com/"]`, kvs["marathon/test/uris"])
	assert.Equal(t, "user", kvs["marathon/test/user"])
	assert.Equal(t, "2015-01-01T00:00:00Z", kvs["marathon/test/version"])
}
