package events

import (
	"encoding/json"
	"strings"

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
	ID     string         `json:"id"`
	Apps   []*apps.App    `json:"apps"`
	Groups []*Deployments `json:"groups"`
}

type CurrentStep struct {
	Actions []*Action `json:"actions"`
}

type Action struct {
	Type  string     `json:"type"`
	AppID apps.AppID `json:"app"`
}

func (d *Deployments) groups() []*Deployments {
	if d.Groups != nil {
		return d.Groups
	}
	return []*Deployments{}
}

func (d *Deployments) apps() []*apps.App {
	if d.Apps != nil {
		return d.Apps
	}
	return []*apps.App{}
}

func (d *DeploymentEvent) originalDeployments() *Deployments {
	if d.Plan != nil {
		if d.Plan.Original != nil {
			return d.Plan.Original
		}
	}
	return &Deployments{}
}

func (d *DeploymentEvent) targetDeployments() *Deployments {
	if d.Plan != nil {
		if d.Plan.Target != nil {
			return d.Plan.Target
		}
	}
	return &Deployments{}
}

func (d *DeploymentEvent) actions() []*Action {
	if d.CurrentStep != nil {
		if d.CurrentStep.Actions != nil {
			return d.CurrentStep.Actions
		}
	}
	return []*Action{}
}

func (d *DeploymentEvent) StoppedConsulApps() []*apps.App {
	return d.consulAppsWithActionPerformed(d.originalDeployments(), "StopApplication")
}

func (d *DeploymentEvent) RestartedConsulApps() []*apps.App {
	return d.consulAppsWithActionPerformed(d.originalDeployments(), "RestartApplication")
}

func (d *DeploymentEvent) RenamedConsulApps() []*apps.App {
	original := d.consulAppsWithActionPerformed(d.originalDeployments(), "RestartApplication")
	renamedApps := []*apps.App{}
	if len(original) > 0 {

		target := d.consulAppsWithActionPerformed(d.targetDeployments(), "RestartApplication")
		originalMap := d.appsMap(original)
		targetMap := d.appsMap(target)
		for id, originalApp := range originalMap {
			targetApp, ok := targetMap[id]
			if !ok || originalApp.ConsulName() != targetApp.ConsulName() {
				renamedApps = append(renamedApps, originalApp)
			}
		}
	}
	return renamedApps
}

func (d *DeploymentEvent) appsMap(applications []*apps.App) map[apps.AppID]*apps.App {
	result := make(map[apps.AppID]*apps.App)
	for _, app := range applications {
		result[app.ID] = app
	}
	return result
}

func (d *DeploymentEvent) consulAppsWithActionPerformed(deployments *Deployments, actionType string) []*apps.App {
	appIds := make(map[apps.AppID]struct{})
	var exists struct{}

	for _, action := range d.actions() {
		if action.Type == actionType {
			appIds[action.AppID] = exists
		}
	}
	return d.filterConsulApps(d.findAppsInDeploymentsGroup(appIds, deployments))
}

func (d *DeploymentEvent) filterConsulApps(allApps []*apps.App) []*apps.App {
	filtered := []*apps.App{}
	for _, app := range allApps {
		if app.IsConsulApp() {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

func (d *DeploymentEvent) findAppsInDeploymentsGroup(appIds map[apps.AppID]struct{}, deployment *Deployments) []*apps.App {
	foundApps := []*apps.App{}
	filteredAppIds := deployment.filterCurrentGroupAppIds(appIds)

	foundInCurrentGroup := d.findAppsInCurrentDeploymentGroupApps(filteredAppIds, deployment)
	for _, app := range foundInCurrentGroup {
		foundApps = append(foundApps, app)
	}

	foundInChildGroups := d.findAppsInDeploymentChildGroups(filteredAppIds, deployment)
	for _, app := range foundInChildGroups {
		foundApps = append(foundApps, app)
	}
	return foundApps
}

func (d *DeploymentEvent) findAppsInCurrentDeploymentGroupApps(appIds map[apps.AppID]struct{}, deployment *Deployments) []*apps.App {
	foundApps := []*apps.App{}
	searchForCount := len(appIds)

	for _, app := range deployment.apps() {
		if searchForCount < 1 {
			break
		}
		if _, ok := appIds[app.ID]; ok {
			foundApps = append(foundApps, app)
			searchForCount--
		}
	}
	return foundApps
}

func (d *DeploymentEvent) findAppsInDeploymentChildGroups(appIds map[apps.AppID]struct{}, deployment *Deployments) []*apps.App {
	foundApps := []*apps.App{}
	searchForCount := len(appIds)

	for _, group := range deployment.groups() {
		if searchForCount < 1 {
			break
		}
		foundInChildGroup := d.findAppsInDeploymentsGroup(appIds, group)
		for _, app := range foundInChildGroup {
			foundApps = append(foundApps, app)
		}
		searchForCount -= len(foundInChildGroup)
	}
	return foundApps
}

func (d *Deployments) filterCurrentGroupAppIds(appIds map[apps.AppID]struct{}) map[apps.AppID]struct{} {
	filteredAppIds := make(map[apps.AppID]struct{})
	var exists struct{}

	for appID := range appIds {
		if strings.HasPrefix(appID.String(), d.ID) {
			filteredAppIds[appID] = exists
		}
	}
	return filteredAppIds
}

func ParseDeploymentEvent(jsonBlob []byte) (*DeploymentEvent, error) {
	deploymentInfo := &DeploymentEvent{}
	err := json.Unmarshal(jsonBlob, deploymentInfo)
	return deploymentInfo, err
}
