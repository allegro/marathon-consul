package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseAPIPostEvent(t *testing.T) {
	input := []byte(`{
	"eventType": "api_post_event",
	"timestamp": "2014-03-01T23:29:30.158Z",
	"clientIp": "0:0:0:0:0:0:0:1",
	"uri": "/v2/apps/my-app",
	"appDefinition": {
		"args": [],
		"backoffFactor": 1.15,
		"backoffSeconds": 1,
		"cmd": "sleep 30",
		"constraints": [],
		"container": null,
		"cpus": 0.2,
		"dependencies": [],
		"disk": 0.0,
		"env": {},
		"executor": "",
		"healthChecks": [],
		"id": "/my-app",
		"instances": 2,
		"labels": {},
		"mem": 32.0,
		"ports": [10001],
		"requirePorts": false,
		"storeUrls": [],
		"upgradeStrategy": {
				"minimumHealthCapacity": 1.0
		},
		"uris": [],
		"user": null,
		"version": "2014-09-09T05:57:50.866Z"
	}
}`)
	output, err := ParseApps(input)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(output))
	assert.Equal(t, "/my-app", output[0].ID)
}

func TestParseDeploymentInfoEvent(t *testing.T) {
	input := []byte(`{
	"eventType": "deployment_info",
	"timestamp": "2014-03-01T23:29:30.158Z",
	"plan": {
		"id": "867ed450-f6a8-4d33-9b0e-e11c5513990b",
		"original": {
			"apps": [],
			"dependencies": [],
			"groups": [],
			"id": "/",
			"version": "2014-09-09T06:30:49.667Z"
		},
		"target": {
			"apps": [
				{
					"args": [],
					"backoffFactor": 1.15,
					"backoffSeconds": 1,
					"cmd": "sleep 30",
					"constraints": [],
					"container": null,
					"cpus": 0.2,
					"dependencies": [],
					"disk": 0.0,
					"env": {},
					"executor": "",
					"healthChecks": [],
					"id": "/my-app",
					"instances": 2,
					"labels": {},
					"mem": 32.0,
					"ports": [10001],
					"requirePorts": false,
					"storeUrls": [],
					"upgradeStrategy": {
							"minimumHealthCapacity": 1.0
					},
					"uris": [],
					"user": null,
					"version": "2014-09-09T05:57:50.866Z"
				}
			],
			"dependencies": [],
			"groups": [],
			"id": "/",
			"version": "2014-09-09T05:57:50.866Z"
		},
		"steps": [
			{
				"action": "ScaleApplication",
				"app": "/my-app"
			}
		],
		"version": "2014-03-01T23:24:14.846Z"
	},
	"currentStep": {
		"action": "ScaleApplication",
		"app": "/my-app"
	}
}`)
	output, err := ParseApps(input)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(output))
	assert.Equal(t, "/my-app", output[0].ID)
}
