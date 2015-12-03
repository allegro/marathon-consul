package tasks

import (
	"encoding/json"
)

type Task struct {
	Timestamp          string              `json:"timestamp"`
	SlaveID            string              `json:"slaveId"`
	ID                 string              `json:"id"`
	TaskStatus         string              `json:"taskStatus"`
	AppID              string              `json:"appId"`
	Host               string              `json:"host"`
	Ports              []int               `json:"ports"`
	Version            string              `json:"version"`
	HealthCheckResults []HealthCheckResult `json:"healthCheckResults"`
}

type HealthCheckResult struct {
	Alive bool `json:"alive"`
}

func ParseTask(event []byte) (*Task, error) {
	task := &Task{}
	err := json.Unmarshal(event, task)
	return task, err
}
