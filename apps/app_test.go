package apps

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var testApp = App{
	Args:           []string{"arg"},
	BackoffFactor:  0.5,
	BackoffSeconds: 1,
	Cmd:            "command",
	Constraints:    [][]string{[]string{"HOSTNAME", "unique"}},
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

func TestAppKey(t *testing.T) {
	t.Parallel()

	app := &App{ID: "a/b/c"}
	assert.Equal(t, app.Key(), "a-b-c")
}

func TestAppKV(t *testing.T) {
	t.Parallel()

	jsonified, err := json.Marshal(testApp)
	assert.Nil(t, err)

	kv := testApp.KV()
	assert.Equal(t, kv.Key, testApp.Key())
	assert.Equal(t, kv.Value, jsonified)
}

func ParseApps(event []byte) (apps []*App, err error) {
	if strings.Index(string(event), "api_post_event") != -1 {
		container := APIPostEvent{}
		err = json.Unmarshal(event, &container)
		if err != nil {
			return nil, err
		}

		apps = []*App{&container.App}
	} else if strings.Index(string(event), "deployment_info") != -1 {
		container := DeploymentInfoEvent{}
		err = json.Unmarshal(event, &container)
		if err != nil {
			return nil, err
		}

		apps = container.Plan.Target.Apps
	} else if strings.Index(string(event), "app_terminated_event") != -1 {
		container := AppTerminatedEvent{}
		err = json.Unmarshal(event, &container)
		if err != nil {
			return nil, err
		}

		apps = []*App{&App{ID: container.AppID}}
	}

	if len(apps) == 0 {
		err = ErrNoApps
	}

	return apps, err
}
