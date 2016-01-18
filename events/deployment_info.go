package events

import (
	"encoding/json"
	"github.com/allegro/marathon-consul/apps"
)

type DeploymentEvent struct {
	Type        string       `json:"eventType"`
	Plan        *Plan        `json:"plan"`
	CurrentStep *CurrentStep `json:"currentStep"`
}

type Plan struct {
	Original *Deployments `json:"original"`
	Target   *Deployments `json:"target"`
}

type Deployments struct {
	Apps []*apps.App `json:"apps"`
}

type CurrentStep struct {
	Actions []*Action `json:"actions"`
}

type Action struct {
	Type  string     `json:"type"`
	AppId apps.AppId `json:"app"`
}

func (d *DeploymentEvent) OriginalApps() []*apps.App {
	if d.Plan != nil {
		if d.Plan.Original != nil {
			if d.Plan.Original.Apps != nil {
				return d.Plan.Original.Apps
			}
		}
	}
	return []*apps.App{}
}

func (d *DeploymentEvent) TargetApps() []*apps.App {
	if d.Plan != nil {
		if d.Plan.Target != nil {
			if d.Plan.Target.Apps != nil {
				return d.Plan.Target.Apps
			}
		}
	}
	return []*apps.App{}
}

func (d *DeploymentEvent) Actions() []*Action {
	if d.CurrentStep != nil {
		if d.CurrentStep.Actions != nil {
			return d.CurrentStep.Actions
		}
	}
	return []*Action{}
}

func (d *DeploymentEvent) StoppedConsulApps() []*apps.App {
	return d.consulAppsWithActionPerformed(d.OriginalApps(), "StopApplication")
}

func (d *DeploymentEvent) RestartedConsulApps() []*apps.App {
	return d.consulAppsWithActionPerformed(d.OriginalApps(), "RestartApplication")
}

func (d *DeploymentEvent) RenamedConsulApps() []*apps.App {
	original := d.consulAppsWithActionPerformed(d.OriginalApps(), "RestartApplication")
	renamedApps := []*apps.App{}
	if len(original) > 0 {

		target := d.consulAppsWithActionPerformed(d.TargetApps(), "RestartApplication")
		originalMap := d.appsMap(original)
		targetMap := d.appsMap(target)
		for id, originalApp := range originalMap {
			targetApp, ok := targetMap[id]
			if !ok || originalApp.ConsulServiceName() != targetApp.ConsulServiceName() {
				renamedApps = append(renamedApps, originalApp)
			}
		}
	}
	return renamedApps
}

func (d *DeploymentEvent) appsMap(applications []*apps.App) map[apps.AppId]*apps.App {
	result := make(map[apps.AppId]*apps.App)
	for _, app := range applications {
		result[app.ID] = app
	}
	return result
}

func (d *DeploymentEvent) consulAppsWithActionPerformed(allApps []*apps.App, actionType string) []*apps.App {
	foundApps := []*apps.App{}
	foundAppIdSet := make(map[apps.AppId]struct{})
	var exists struct{}

	for _, action := range d.Actions() {
		if action.Type == actionType {
			foundAppIdSet[action.AppId] = exists
		}
	}

	if len(foundAppIdSet) > 0 {
		for _, app := range allApps {
			if _, ok := foundAppIdSet[app.ID]; ok && app.IsConsulApp() {
				foundApps = append(foundApps, app)
			}
		}
	}
	return foundApps
}

func ParseDeploymentEvent(jsonBlob []byte) (*DeploymentEvent, error) {
	deploymentInfo := &DeploymentEvent{}
	err := json.Unmarshal(jsonBlob, deploymentInfo)
	return deploymentInfo, err
}
