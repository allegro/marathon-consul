package events

import (
	"github.com/allegro/marathon-consul/apps"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestParseDeploymentInfo(t *testing.T) {
	t.Parallel()
	// given
	blob, _ := ioutil.ReadFile("deployment_info.json")

	// when
	deploymentInfo, err := ParseDeploymentEvent(blob)
	action := deploymentInfo.actions()[0]
	app := deploymentInfo.originalDeployments().apps()[0]

	// then
	assert.NoError(t, err)
	assert.Equal(t, "StopApplication", action.Type)
	assert.Equal(t, "/internalName", action.AppId.String())
	assert.Equal(t, "/internalName", app.ID.String())
	assert.Equal(t, "consulName", app.Labels["consul"])
}

func TestParseDeploymentStepSuccess(t *testing.T) {
	t.Parallel()
	// given
	blob, _ := ioutil.ReadFile("deployment_step_success.json")

	// when
	deploymentInfo, err := ParseDeploymentEvent(blob)
	action := deploymentInfo.actions()[0]
	app := deploymentInfo.RestartedConsulApps()[0]

	// then
	assert.NoError(t, err)
	assert.Equal(t, "RestartApplication", action.Type)
	assert.Equal(t, "/a", action.AppId.String())
	assert.Equal(t, "/a", app.ID.String())
	assert.Equal(t, "b", app.Labels["consul"])
}

func TestParseDeploymentStepSuccessWithGroups(t *testing.T) {
	t.Parallel()
	// given
	blob, _ := ioutil.ReadFile("deployment_step_success_with_groups.json")

	// when
	deploymentInfo, err := ParseDeploymentEvent(blob)
	action := deploymentInfo.actions()[0]
	app := deploymentInfo.RestartedConsulApps()[0]

	// then
	assert.NoError(t, err)
	assert.Equal(t, "RestartApplication", action.Type)
	assert.Equal(t, "/com.example/things/something", action.AppId.String())
	assert.Equal(t, "/com.example/things/something", app.ID.String())
	assert.Equal(t, "something", app.Labels["consul"])
}

func TestOriginalDeployments_OnEmpty(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{}

	// when
	deployments := deploymentInfo.originalDeployments()

	// then
	assert.NotNil(t, deployments)
}

func TestTargetDeployments_OnEmpty(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{}

	// when
	deployments := deploymentInfo.targetDeployments()

	// then
	assert.NotNil(t, deployments)
}

func TestActions_OnEmpty(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{}

	// when
	actions := deploymentInfo.actions()

	// then
	assert.Len(t, actions, 0)
}

func TestStoppedConsulApps(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1"},
					&apps.App{ID: "app2", Labels: map[string]string{"consul": ""}},
					&apps.App{ID: "app3", Labels: map[string]string{"consul": ""}},
					&apps.App{ID: "app4"},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "StartApplication", AppId: "app1"},
				&Action{Type: "StopApplication", AppId: "app2"},
				&Action{Type: "StopApplication", AppId: "app3"},
				&Action{Type: "StopApplication", AppId: "app4"},
			},
		},
	}

	// when
	stoppedApps := deploymentInfo.StoppedConsulApps()

	// then
	assert.Len(t, stoppedApps, 2)
	assert.Contains(t, stoppedApps, deploymentInfo.originalDeployments().apps()[1])
	assert.Contains(t, stoppedApps, deploymentInfo.originalDeployments().apps()[2])
}

func TestRestartedConsulApps(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1"},
					&apps.App{ID: "app2", Labels: map[string]string{"consul": ""}},
					&apps.App{ID: "app3", Labels: map[string]string{"consul": "someName"}},
					&apps.App{ID: "app4"},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "StartApplication", AppId: "app1"},
				&Action{Type: "RestartApplication", AppId: "app2"},
				&Action{Type: "RestartApplication", AppId: "app3"},
				&Action{Type: "RestartApplication", AppId: "app4"},
			},
		},
	}

	// when
	restartedApps := deploymentInfo.RestartedConsulApps()

	// then
	assert.Len(t, restartedApps, 2)
	assert.Contains(t, restartedApps, deploymentInfo.originalDeployments().apps()[1])
	assert.Contains(t, restartedApps, deploymentInfo.originalDeployments().apps()[2])
}

func TestStoppedConsulApps_NoStoppedApps(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1"},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "StartApplication", AppId: "app1"},
			},
		},
	}

	// when
	stoppedApps := deploymentInfo.StoppedConsulApps()

	// then
	assert.Len(t, stoppedApps, 0)
}

func TestStoppedConsulApps_OnEmpty(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{}

	// when
	stoppedApps := deploymentInfo.StoppedConsulApps()

	// then
	assert.Len(t, stoppedApps, 0)
}

func TestRenamedConsulApps(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1", Labels: map[string]string{"consul": "same"}},
					&apps.App{ID: "app2"},
					&apps.App{ID: "app3", Labels: map[string]string{"consul": "customName"}},
				},
			},
			Target: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1", Labels: map[string]string{"consul": "same"}},
					&apps.App{ID: "app2"},
					&apps.App{ID: "app3", Labels: map[string]string{"consul": "newCustomName"}},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "RestartApplication", AppId: "app1"},
				&Action{Type: "StartApplication", AppId: "app2"},
				&Action{Type: "RestartApplication", AppId: "app3"},
			},
		},
	}

	// when
	renamedApps := deploymentInfo.RenamedConsulApps()

	// then
	assert.Len(t, renamedApps, 1)
	assert.Contains(t, renamedApps, deploymentInfo.originalDeployments().apps()[2])
}

func TestRenamedConsulApps_OnEmptyPlan(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "RestartApplication", AppId: "app2"},
				&Action{Type: "StartApplication", AppId: "app1"},
				&Action{Type: "RestartApplication", AppId: "app3"},
			},
		},
	}

	// when
	renamedApps := deploymentInfo.RenamedConsulApps()

	// then
	assert.Len(t, renamedApps, 0)
}

func TestRenamedConsulApps_OnConsulTrueCase(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1"},
					&apps.App{ID: "app2"},
				},
			},
			Target: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1", Labels: map[string]string{"consul": "true"}},
					&apps.App{ID: "app2"},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "RestartApplication", AppId: "app1"},
				&Action{Type: "RestartApplication", AppId: "app2"},
			},
		},
	}

	// when
	renamedApps := deploymentInfo.RenamedConsulApps()

	// then
	assert.Len(t, renamedApps, 0)
}

func TestRenamedConsulApps_OnCustomNameAdded(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1", Labels: map[string]string{"consul": ""}},
					&apps.App{ID: "app2"},
				},
			},
			Target: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1", Labels: map[string]string{"consul": "customName"}},
					&apps.App{ID: "app2"},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "RestartApplication", AppId: "app1"},
				&Action{Type: "StartApplication", AppId: "app2"},
			},
		},
	}

	// when
	renamedApps := deploymentInfo.RenamedConsulApps()

	// then
	assert.Len(t, renamedApps, 1)
	assert.Contains(t, renamedApps, deploymentInfo.originalDeployments().apps()[0])
}

func TestRenamedConsulApps_OnCustomNameAddedToNonConsulApp(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1"},
					&apps.App{ID: "app2"},
				},
			},
			Target: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1", Labels: map[string]string{"consul": "customName"}},
					&apps.App{ID: "app2"},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "RestartApplication", AppId: "app1"},
				&Action{Type: "StartApplication", AppId: "app2"},
			},
		},
	}

	// when
	renamedApps := deploymentInfo.RenamedConsulApps()

	// then
	assert.Len(t, renamedApps, 0)
}

func TestRenamedConsulApps_OnConsulLabelRemoved(t *testing.T) {
	t.Parallel()
	// given
	deploymentInfo := &DeploymentEvent{
		Plan: &Plan{
			Original: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1", Labels: map[string]string{"consul": "customName"}},
					&apps.App{ID: "app2"},
				},
			},
			Target: &Deployments{
				Apps: []*apps.App{
					&apps.App{ID: "app1"},
					&apps.App{ID: "app2"},
				},
			},
		},
		CurrentStep: &CurrentStep{
			Actions: []*Action{
				&Action{Type: "RestartApplication", AppId: "app1"},
				&Action{Type: "StartApplication", AppId: "app2"},
			},
		},
	}

	// when
	renamedApps := deploymentInfo.RenamedConsulApps()

	// then
	assert.Len(t, renamedApps, 1)
	assert.Contains(t, renamedApps, deploymentInfo.originalDeployments().apps()[0])
}
