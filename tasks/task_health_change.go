package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/utils"
)

type TaskHealthChange struct {
	Timestamp  string `json:"timestamp"`
	ID         string `json:"id"`
	TaskStatus string `json:"taskStatus"`
	AppID      string `json:"appId"`
	Version    string `json:"version"`
	Alive      bool   `json:"alive"`
}

func ParseTaskHealthChange(event []byte) (*TaskHealthChange, error) {
	task := &TaskHealthChange{}
	err := json.Unmarshal(event, task)
	return task, err
}

func (task *TaskHealthChange) Key() string {
	return fmt.Sprintf(
		"%s/tasks/%s",
		utils.CleanID(task.AppID),
		task.ID,
	)
}
