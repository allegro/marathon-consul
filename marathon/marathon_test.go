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
    "id":"/tests",
    "cmd":"sleep 10 && python -m SimpleHTTPServer $PORT0",
    "args":null,
    "user":null,
    "env":{

    },
    "instances":1,
    "cpus":0.1,
    "mem":16,
    "disk":0,
    "executor":"",
    "constraints":[

    ],
    "uris":[

    ],
    "storeUrls":[

    ],
    "ports":[
        10002
    ],
    "requirePorts":false,
    "backoffSeconds":1,
    "backoffFactor":1.15,
    "maxLaunchDelaySeconds":3600,
    "container":null,
    "healthChecks":[
        {
            "path":"/",
            "protocol":"HTTP",
            "portIndex":0,
            "gracePeriodSeconds":5,
            "intervalSeconds":60,
            "timeoutSeconds":10,
            "maxConsecutiveFailures":3,
            "ignoreHttp1xx":false
        }
    ],
    "dependencies":[

    ],
    "upgradeStrategy":{
        "minimumHealthCapacity":1,
        "maximumOverCapacity":1
    },
    "labels":{

    },
    "acceptedResourceRoles":null,
    "version":"2015-11-27T12:35:14.601Z",
    "versionInfo":{
        "lastScalingAt":"2015-11-27T12:35:14.601Z",
        "lastConfigChangeAt":"2015-11-27T12:35:14.601Z"
    },
    "tasksStaged":0,
    "tasksRunning":1,
    "tasksHealthy":1,
    "tasksUnhealthy":0,
    "deployments":[

    ],
    "tasks":[
        {
            "id":"tests.a8ad5a76-974c-11e5-a62c-024237193611",
            "host":"localhost",
            "ports":[
                31334
            ],
            "startedAt":"2015-11-30T10:25:21.146Z",
            "stagedAt":"2015-11-30T10:25:20.863Z",
            "version":"2015-11-27T12:35:14.601Z",
            "slaveId":"85e59460-a99e-4f16-b91f-145e0ea595bd-S0",
            "appId":"/tests",
            "healthCheckResults":[
                {
                    "alive":true,
                    "consecutiveFailures":0,
                    "firstSuccess":"2015-11-30T10:26:04.770Z",
                    "lastFailure":null,
                    "lastSuccess":"2015-11-30T12:57:07.784Z",
                    "taskId":"tests.a8ad5a76-974c-11e5-a62c-024237193611"
                }
            ]
        }
    ],
    "lastTaskFailure":{
        "appId":"/tests",
        "host":"c50940.allegrogroup.internal",
        "message":"Reconciliation: Task is unknown",
        "state":"TASK_LOST",
        "taskId":"tests.4ef0c377-9503-11e5-bf51-0242eaaee42f",
        "timestamp":"2015-11-30T10:25:19.623Z",
        "version":"2015-11-27T12:35:14.601Z",
        "slaveId":"20151126-090640-16842879-5050-3336-S0"
    }
}
`)

	m, _ := NewMarathon("localhost:8080", "http", nil)
	app, err := m.ParseApp(appBlob)
	assert.Nil(t, err)
	assert.Equal(t, len(app.Tasks), 1)
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
