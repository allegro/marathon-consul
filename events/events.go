package events

import (
	"github.com/CiscoCloud/marathon-consul/apps"
)

type Event interface {
	Apps() []*apps.App
}

type BaseEvent struct {
	Type string `json:"eventType"`
}

type APIPostEvent struct {
	Type string    `json:"eventType"`
	App  *apps.App `json:"appDefinition"`
}

func (event APIPostEvent) Apps() []*apps.App {
	return []*apps.App{event.App}
}

type DeploymentInfoEvent struct {
	Type string `json:"eventType"`
	Plan struct {
		Target struct {
			Apps []*apps.App `json:"apps"`
		} `json:"target"`
	} `json:"plan"`
	CurrentStep struct {
		Action string `json:"action"`
		App    string `json:"app"`
	} `json:"currentStep"`
}

func (event DeploymentInfoEvent) Apps() []*apps.App {
	return event.Plan.Target.Apps
}

type AppTerminatedEvent struct {
	Type      string `json:"eventType"`
	AppID     string `json:"appId"`
	Timestamp string `json:"timestamp"`
}

func (event AppTerminatedEvent) Apps() []*apps.App {
	return []*apps.App{
		&apps.App{ID: event.AppID},
	}
}
