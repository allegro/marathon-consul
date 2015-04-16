package main

import (
	"encoding/json"
	"errors"
	"strings"
)

var ErrNoApps = errors.New("no apps present in provided JSON")

type APIPostEvent struct {
	Type string `json:"eventType"`
	App  App    `json:"appDefinition"`
}

type DeploymentInfoEvent struct {
	Type string `json:"eventType"`
	Plan struct {
		Target struct {
			Apps []*App `json:"apps"`
		} `json:"target"`
	} `json:"plan"`
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
	}

	if len(apps) == 0 {
		err = ErrNoApps
	}

	return
}
