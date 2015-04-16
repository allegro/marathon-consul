package main

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"strconv"
	"strings"
)

type Docker struct {
	Image      string   `json:"image"`
	Parameters []string `json:"parameters"`
	Privileged bool     `json:"privileged"`
}

func (d *Docker) KVs(app *App) []*api.KVPair {
	params, _ := json.Marshal(d.Parameters)

	return []*api.KVPair{
		&api.KVPair{
			Key:   app.Key("container/docker/image"),
			Value: []byte(d.Image),
		},
		&api.KVPair{
			Key:   app.Key("container/docker/parameters"),
			Value: params,
		},
		&api.KVPair{
			Key:   app.Key("container/docker/privileged"),
			Value: []byte(strconv.FormatBool(d.Privileged)),
		},
	}
}

type Volume struct {
	ContainerPath string `json:"containerPath"`
	HostPath      string `json:"hostPath"`
	Mode          string `json:"mode"`
}

type Container struct {
	Docker  *Docker  `json:"docker"`
	Type    string   `json:"type"`
	Volumes []Volume `json:"volumes"`
}

func (c *Container) KVs(app *App) []*api.KVPair {
	out := c.Docker.KVs(app)
	volumes, _ := json.Marshal(c.Volumes)

	return append(
		out,
		&api.KVPair{Key: app.Key("container/type"), Value: []byte(c.Type)},
		&api.KVPair{Key: app.Key("container/volumes"), Value: volumes},
	)
}

type HealthCheck struct {
	Path                   string `json:"path"`
	PortIndex              int    `json:"portIndex"`
	Protocol               string `json:"protocol"`
	GracePeriodSeconds     int    `json:"gracePeriodSeconds"`
	IntervalSeconds        int    `json:"intervalSeconds"`
	TimeoutSeconds         int    `json:"timeoutSeconds"`
	MaxConsecutiveFailures int    `json:"maxConsecutiveFailures"`
}

type UpgradeStrategy struct {
	MinimumHealthCapacity float64 `json:"minimumHealthCapacity"`
	MaximumOverCapacity   float64 `json:"maximumOverCapacity"`
}

func (u UpgradeStrategy) KVs(app *App) []*api.KVPair {
	return []*api.KVPair{
		&api.KVPair{
			Key:   app.Key("upgradeStrategy/minimumHealthCapacity"),
			Value: []byte(strconv.FormatFloat(u.MinimumHealthCapacity, 'g', -1, 64)),
		},
		&api.KVPair{
			Key:   app.Key("upgradeStrategy/maximumOverCapacity"),
			Value: []byte(strconv.FormatFloat(u.MaximumOverCapacity, 'g', -1, 64)),
		},
	}
}

type App struct {
	Args            []string          `json:"args"`
	BackoffFactor   float64           `json:"backoffFactor"`
	BackoffSeconds  int               `json:"backoffSeconds"`
	Cmd             string            `json:"cmd"`
	Constraints     []string          `json:"constraints"`
	Container       *Container        `json:"container"`
	CPUs            float64           `json:"cpus"`
	Dependencies    []string          `json:"dependencies"`
	Disk            float64           `json:"disk"`
	Env             map[string]string `json:"env"`
	Executor        string            `json:"executor"`
	Labels          map[string]string `json:"labels"`
	HealthChecks    []HealthCheck     `json:"healthChecks"`
	ID              string            `json:"id"`
	Instances       int               `json:"instances"`
	Mem             float64           `json:"mem"`
	Ports           []int             `json:"ports"`
	RequirePorts    bool              `json:"requirePorts"`
	StoreUrls       []string          `json:"storeUrls"`
	UpgradeStrategy UpgradeStrategy   `json:"upgradeStrategy"`
	Uris            []string          `json:"uris"`
	User            string            `json:"user"`
	Version         string            `json:"version"`
}

func (app *App) Key(postfix string) string {
	return fmt.Sprintf(
		"/marathon/%s/%s",
		strings.Trim(app.ID, "/"),
		postfix,
	)
}

func (app *App) KVs() []*api.KVPair {
	// simple containers (lists and maps) are encoded as JSON. This allows us to
	// only add or change keys, and simplifies watches.
	args, _ := json.Marshal(app.Args)
	constraints, _ := json.Marshal(app.Constraints)
	dependencies, _ := json.Marshal(app.Dependencies)
	env, _ := json.Marshal(app.Env)
	healthChecks, _ := json.Marshal(app.HealthChecks)
	labels, _ := json.Marshal(app.Labels)
	ports, _ := json.Marshal(app.Ports)
	storeUrls, _ := json.Marshal(app.StoreUrls)
	uris, _ := json.Marshal(app.Uris)

	kvs := []*api.KVPair{
		// containers
		&api.KVPair{Key: app.Key("args"), Value: args},
		&api.KVPair{Key: app.Key("constraints"), Value: constraints},
		&api.KVPair{Key: app.Key("dependencies"), Value: dependencies},
		&api.KVPair{Key: app.Key("env"), Value: env},
		&api.KVPair{Key: app.Key("healthChecks"), Value: healthChecks},
		&api.KVPair{Key: app.Key("labels"), Value: labels},
		&api.KVPair{Key: app.Key("ports"), Value: ports},
		&api.KVPair{Key: app.Key("storeUrls"), Value: storeUrls},
		&api.KVPair{Key: app.Key("uris"), Value: uris},

		// "scalar" values
		&api.KVPair{
			Key:   app.Key("backoffFactor"),
			Value: []byte(strconv.FormatFloat(app.BackoffFactor, 'g', -1, 64)),
		},
		&api.KVPair{
			Key:   app.Key("backoffSeconds"),
			Value: []byte(strconv.Itoa(app.BackoffSeconds)),
		},
		&api.KVPair{
			Key:   app.Key("cmd"),
			Value: []byte(app.Cmd),
		},
		&api.KVPair{
			Key:   app.Key("cpus"),
			Value: []byte(strconv.FormatFloat(app.CPUs, 'g', -1, 64)),
		},
		&api.KVPair{
			Key:   app.Key("disk"),
			Value: []byte(strconv.FormatFloat(app.Disk, 'g', -1, 64)),
		},
		&api.KVPair{
			Key:   app.Key("executor"),
			Value: []byte(app.Executor),
		},
		&api.KVPair{
			Key:   app.Key("id"),
			Value: []byte(app.ID),
		},
		&api.KVPair{
			Key:   app.Key("instances"),
			Value: []byte(strconv.Itoa(app.Instances)),
		},
		&api.KVPair{
			Key:   app.Key("mem"),
			Value: []byte(strconv.FormatFloat(app.Mem, 'g', -1, 64)),
		},
		&api.KVPair{
			Key:   app.Key("requirePorts"),
			Value: []byte(strconv.FormatBool(app.RequirePorts)),
		},
		&api.KVPair{
			Key:   app.Key("user"),
			Value: []byte(app.User),
		},
		&api.KVPair{
			Key:   app.Key("version"),
			Value: []byte(app.Version),
		},
	}

	if app.Container != nil {
		kvs = append(kvs, app.Container.KVs(app)...)
	}
	kvs = append(kvs, app.UpgradeStrategy.KVs(app)...)

	return kvs
}
