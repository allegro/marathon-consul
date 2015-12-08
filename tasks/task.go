package tasks

import (
	"encoding/json"
)

type Task struct {
	ID                 string              `json:"id"`
	TaskStatus         string              `json:"taskStatus"`
	AppID              string              `json:"appId"`
	Host               string              `json:"host"`
	Ports              []int               `json:"ports"`
	HealthCheckResults []HealthCheckResult `json:"healthCheckResults"`
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
