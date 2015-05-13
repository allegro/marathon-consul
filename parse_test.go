package main

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseAPIPostEvent(t *testing.T) {
	t.Parallel()

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
	assert.True(t, output[0].Active)
	assert.Equal(t, 0, len(output[0].Args))
	assert.Equal(t, 1.15, output[0].BackoffFactor)
	assert.Equal(t, 1, output[0].BackoffSeconds)
	assert.Equal(t, "sleep 30", output[0].Cmd)
	assert.Equal(t, 0, len(output[0].Constraints))
	assert.Nil(t, output[0].Container)
	assert.Equal(t, 0.2, output[0].CPUs)
	assert.Equal(t, 0, len(output[0].Dependencies))
	assert.Equal(t, 0.0, output[0].Disk)
	assert.Equal(t, map[string]string{}, output[0].Env)
	assert.Equal(t, "", output[0].Executor)
	assert.Equal(t, 0, len(output[0].HealthChecks))
	assert.Equal(t, "/my-app", output[0].ID)
	assert.Equal(t, 2, output[0].Instances)
	assert.Equal(t, map[string]string{}, output[0].Labels)
	assert.Equal(t, 32.0, output[0].Mem)
	assert.Equal(t, []int{10001}, output[0].Ports)
	assert.False(t, output[0].RequirePorts)
	assert.Equal(t, 1.0, output[0].UpgradeStrategy.MinimumHealthCapacity)
	assert.Equal(t, 0, len(output[0].Uris))
	assert.Equal(t, "", output[0].User)
	assert.Equal(t, "2014-09-09T05:57:50.866Z", output[0].Version)
}

func TestParseDeploymentInfoEvent(t *testing.T) {
	t.Parallel()

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
	assert.True(t, output[0].Active)
	assert.Equal(t, 0, len(output[0].Args))
	assert.Equal(t, 1.15, output[0].BackoffFactor)
	assert.Equal(t, 1, output[0].BackoffSeconds)
	assert.Equal(t, "sleep 30", output[0].Cmd)
	assert.Equal(t, 0, len(output[0].Constraints))
	assert.Nil(t, output[0].Container)
	assert.Equal(t, 0.2, output[0].CPUs)
	assert.Equal(t, 0, len(output[0].Dependencies))
	assert.Equal(t, 0.0, output[0].Disk)
	assert.Equal(t, map[string]string{}, output[0].Env)
	assert.Equal(t, "", output[0].Executor)
	assert.Equal(t, 0, len(output[0].HealthChecks))
	assert.Equal(t, "/my-app", output[0].ID)
	assert.Equal(t, 2, output[0].Instances)
	assert.Equal(t, map[string]string{}, output[0].Labels)
	assert.Equal(t, 32.0, output[0].Mem)
	assert.Equal(t, []int{10001}, output[0].Ports)
	assert.False(t, output[0].RequirePorts)
	assert.Equal(t, 1.0, output[0].UpgradeStrategy.MinimumHealthCapacity)
	assert.Equal(t, 0, len(output[0].Uris))
	assert.Equal(t, "", output[0].User)
	assert.Equal(t, "2014-09-09T05:57:50.866Z", output[0].Version)
}

func TestParseDeploymentInfoEventStop(t *testing.T) {
	t.Parallel()

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
		"action": "StopApplication",
		"app": "/my-app"
	}
}`)
	output, err := ParseApps(input)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(output))
	assert.False(t, output[0].Active)
	assert.Equal(t, 0, len(output[0].Args))
	assert.Equal(t, 1.15, output[0].BackoffFactor)
	assert.Equal(t, 1, output[0].BackoffSeconds)
	assert.Equal(t, "sleep 30", output[0].Cmd)
	assert.Equal(t, 0, len(output[0].Constraints))
	assert.Nil(t, output[0].Container)
	assert.Equal(t, 0.2, output[0].CPUs)
	assert.Equal(t, 0, len(output[0].Dependencies))
	assert.Equal(t, 0.0, output[0].Disk)
	assert.Equal(t, map[string]string{}, output[0].Env)
	assert.Equal(t, "", output[0].Executor)
	assert.Equal(t, 0, len(output[0].HealthChecks))
	assert.Equal(t, "/my-app", output[0].ID)
	assert.Equal(t, 2, output[0].Instances)
	assert.Equal(t, map[string]string{}, output[0].Labels)
	assert.Equal(t, 32.0, output[0].Mem)
	assert.Equal(t, []int{10001}, output[0].Ports)
	assert.False(t, output[0].RequirePorts)
	assert.Equal(t, 1.0, output[0].UpgradeStrategy.MinimumHealthCapacity)
	assert.Equal(t, 0, len(output[0].Uris))
	assert.Equal(t, "", output[0].User)
	assert.Equal(t, "2014-09-09T05:57:50.866Z", output[0].Version)
}

func TestParseEmpty(t *testing.T) {
	t.Parallel()

	input := []byte(`{"eventType":"deployment_info"}`)
	output, err := ParseApps(input)

	assert.Equal(t, errors.New("no apps present in provided JSON"), err)
	assert.Equal(t, 0, len(output))
}
