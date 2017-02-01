package apps

import (
	"encoding/json"
	"strings"
)

type Task struct {
	ID                 TaskID              `json:"id"`
	TaskStatus         string              `json:"taskStatus"`
	AppID              AppID               `json:"appId"`
	Host               string              `json:"host"`
	Ports              []int               `json:"ports"`
	HealthCheckResults []HealthCheckResult `json:"healthCheckResults"`
}

// Marathon Task ID
// Usually in the form of AppId.uuid with '/' replaced with '_'
type TaskID string

func (id TaskID) String() string {
	return string(id)
}

func (id TaskID) AppID() AppID {
	index := strings.LastIndex(id.String(), ".")
	return AppID("/" + strings.Replace(id.String()[0:index], "_", "/", -1))
}

type HealthCheckResult struct {
	Alive bool `json:"alive"`
}

type TasksResponse struct {
	Tasks []*Task `json:"tasks"`
}

func ParseTasks(jsonBlob []byte) ([]*Task, error) {
	tasks := &TasksResponse{}
	err := json.Unmarshal(jsonBlob, tasks)

	return tasks.Tasks, err
}

func ParseTask(event []byte) (*Task, error) {
	task := &Task{}
	err := json.Unmarshal(event, task)
	return task, err
}

func (t Task) IsHealthy() bool {
	if len(t.HealthCheckResults) < 1 {
		return false
	}
	register := true
	for _, healthCheckResult := range t.HealthCheckResults {
		register = register && healthCheckResult.Alive
	}
	return register
}
