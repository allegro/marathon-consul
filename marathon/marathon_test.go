package marathon

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUrl(t *testing.T) {
	t.Parallel()

	m, _ := NewMarathon("localhost:8080", "http", nil)
	url := m.Url("/v2/apps")

	assert.Equal(t, url, "http://localhost:8080/v2/apps")
}

func TestParseApps(t *testing.T) {
	t.Parallel()

	appBlob := []byte(`{
    "apps": [{
        "args": null,
        "backoffFactor": 1.15,
        "backoffSeconds": 1,
        "maxLaunchDelaySeconds": 3600,
        "cmd": "python3 -m http.server 8080",
        "constraints": [],
        "container": {
            "docker": {
                "image": "python:3",
                "network": "BRIDGE",
                "portMappings": [
                    {"containerPort": 8080, "hostPort": 0, "servicePort": 9000, "protocol": "tcp"},
                    {"containerPort": 161, "hostPort": 0, "protocol": "udp"}
                ]
            },
            "type": "DOCKER",
            "volumes": []
        },
        "cpus": 0.5,
        "dependencies": [],
        "deployments": [],
        "disk": 0.0,
        "env": {},
        "executor": "",
        "healthChecks": [{
            "command": null,
            "gracePeriodSeconds": 5,
            "intervalSeconds": 20,
            "maxConsecutiveFailures": 3,
            "path": "/",
            "portIndex": 0,
            "protocol": "HTTP",
            "timeoutSeconds": 20
        }],
        "id": "/bridged-webapp",
        "instances": 2,
        "mem": 64.0,
        "ports": [10000, 10001],
        "requirePorts": false,
        "storeUrls": [],
        "tasksRunning": 2,
        "tasksHealthy": 2,
        "tasksUnhealthy": 0,
        "tasksStaged": 0,
        "upgradeStrategy": {"minimumHealthCapacity": 1.0},
        "uris": [],
        "user": null,
        "version": "2014-09-25T02:26:59.256Z",
		"tasks": [
			{
				"appId": "/test",
				"host": "192.168.2.114",
				"id": "test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8",
				"ports": [31315],
				"stagedAt": "2015-06-24T14:57:06.353Z",
				"startedAt": "2015-06-24T14:57:06.466Z",
				"version": "2015-06-24T14:56:57.466Z",
				"healthCheckResults":[
					{
						"alive":true,
						"consecutiveFailures":0,
						"firstSuccess":"2015-11-28T18:21:11.957Z",
						"lastFailure":null,
						"lastSuccess":"2015-11-30T10:08:19.477Z",
						"taskId":"bridged-webapp.a9b051fb-95fc-11e5-9571-02818b42970e"
					}
				]
			},
			{
				"appId": "/test",
				"host": "192.168.2.114",
				"id": "test.4453212c-1a81-11e5-bdb6-e6cb6734eaf8",
				"ports": [31797],
				"stagedAt": "2015-06-24T14:57:00.474Z",
				"startedAt": "2015-06-24T14:57:00.611Z",
				"version": "2015-06-24T14:56:57.466Z"
			}
		]
    }
]}
`)

	m, _ := NewMarathon("localhost:8080", "http", nil)
	apps, err := m.ParseApps(appBlob)
	assert.Nil(t, err)
	assert.Equal(t, len(apps), 1)
}

func TestParseApp(t *testing.T) {
	t.Parallel()

	appBlob := []byte(`{
	"app": {
		"id": "/myapp",
		"cmd": "env && python -m SimpleHTTPServer $PORT0",
		"args": null,
		"user": null,
		"env": {},
		"instances": 2,
		"cpus": 0.1,
		"mem": 32.0,
		"disk": 0.0,
		"executor": "",
		"constraints": [],
		"uris": [],
		"storeUrls": [],
		"ports": [10002, 1, 2, 3],
		"requirePorts": false,
		"backoffSeconds": 1,
		"backoffFactor": 1.15,
		"maxLaunchDelaySeconds": 3600,
		"container": null,
		"healthChecks": [{
			"path": "/",
			"protocol": "HTTP",
			"portIndex": 0,
			"gracePeriodSeconds": 10,
			"intervalSeconds": 5,
			"timeoutSeconds": 10,
			"maxConsecutiveFailures": 3,
			"ignoreHttp1xx": false
		}],
		"dependencies": [],
		"upgradeStrategy": {
			"minimumHealthCapacity": 1.0,
			"maximumOverCapacity": 1.0
		},
		"labels": {
			"consul": "true",
			"public": "tag"
		},
		"version": "2015-12-01T10:03:32.003Z",
		"tasksStaged": 0,
		"tasksRunning": 2,
		"tasksHealthy": 2,
		"tasksUnhealthy": 0,
		"deployments": [],
		"tasks": [{
			"id": "myapp.cc49ccc1-9812-11e5-a06e-56847afe9799",
			"host": "10.141.141.10",
			"ports": [31678, 31679, 31680, 31681],
			"startedAt": "2015-12-01T10:03:40.966Z",
			"stagedAt": "2015-12-01T10:03:40.890Z",
			"version": "2015-12-01T10:03:32.003Z",
			"appId": "/myapp",
			"healthCheckResults": [{
				"alive": true,
				"consecutiveFailures": 0,
				"firstSuccess": "2015-12-01T10:03:42.324Z",
				"lastFailure": null,
				"lastSuccess": "2015-12-01T10:03:42.324Z",
				"taskId": "myapp.cc49ccc1-9812-11e5-a06e-56847afe9799"
			}]
		}, {
			"id": "myapp.c8b449f0-9812-11e5-a06e-56847afe9799",
			"host": "10.141.141.10",
			"ports": [31307, 31308, 31309, 31310],
			"startedAt": "2015-12-01T10:03:34.945Z",
			"stagedAt": "2015-12-01T10:03:34.877Z",
			"version": "2015-12-01T10:03:32.003Z",
			"appId": "/myapp",
			"healthCheckResults": [{
				"alive": true,
				"consecutiveFailures": 0,
				"firstSuccess": "2015-12-01T10:03:37.313Z",
				"lastFailure": null,
				"lastSuccess": "2015-12-01T10:03:42.337Z",
				"taskId": "myapp.c8b449f0-9812-11e5-a06e-56847afe9799"
			}]
		}],
		"lastTaskFailure": null
	}
}`)

	m, _ := NewMarathon("localhost:8080", "http", nil)
	app, err := m.ParseApp(appBlob)
	assert.Nil(t, err)
	assert.Equal(t, len(app.Tasks), 2)
	assert.Equal(t, len(app.HealthChecks), 1)
	assert.Equal(t, "true", app.Labels["consul"])
	assert.Equal(t, "tag", app.Labels["public"])
}

func TestParseTasks(t *testing.T) {
	t.Parallel()

	tasksBlob := []byte(`{
    "tasks": [
        {
            "appId": "/test",
            "host": "192.168.2.114",
            "id": "test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8",
            "ports": [31315],
            "stagedAt": "2015-06-24T14:57:06.353Z",
            "startedAt": "2015-06-24T14:57:06.466Z",
            "version": "2015-06-24T14:56:57.466Z",
            "healthCheckResults":[
				{
					"alive":true,
					"consecutiveFailures":0,
					"firstSuccess":"2015-11-28T18:21:11.957Z",
					"lastFailure":null,
					"lastSuccess":"2015-11-30T10:08:19.477Z",
					"taskId":"bridged-webapp.a9b051fb-95fc-11e5-9571-02818b42970e"
				}
			]
        },
        {
            "appId": "/test",
            "host": "192.168.2.114",
            "id": "test.4453212c-1a81-11e5-bdb6-e6cb6734eaf8",
            "ports": [31797],
            "stagedAt": "2015-06-24T14:57:00.474Z",
            "startedAt": "2015-06-24T14:57:00.611Z",
            "version": "2015-06-24T14:56:57.466Z"
        }
    ]
}
`)

	m, _ := NewMarathon("localhost:8080", "http", nil)
	tasks, err := m.ParseTasks(tasksBlob)
	assert.Nil(t, err)
	assert.Equal(t, len(tasks), 2)
}
