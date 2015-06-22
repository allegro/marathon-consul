package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/utils"
	"github.com/hashicorp/consul/api"
)

type Task struct {
	Timestamp  string `json:"timestamp"`
	SlaveID    string `json:"slaveId"`
	TaskID     string `json:"taskId"`
	TaskStatus string `json:"taskStatus"`
	AppID      string `json:"appId"`
	Host       string `json:"host"`
	Ports      []int  `json:"ports"`
	Version    string `json:"version"`
}

func ParseTask(event []byte) (*Task, error) {
	task := &Task{}
	err := json.Unmarshal(event, task)
	return task, err
}

func (task *Task) Key() string {
	return fmt.Sprintf(
		"%s/tasks/%s",
		utils.CleanID(task.AppID),
		task.TaskID,
	)
}

func (task *Task) KV() *api.KVPair {
	serialized, _ := json.Marshal(task)

	return &api.KVPair{
		Key:   task.Key(),
		Value: serialized,
	}
}
