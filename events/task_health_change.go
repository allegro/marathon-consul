package events

import (
	"encoding/json"
	"github.com/allegro/marathon-consul/tasks"
)

type TaskHealthChange struct {
	Timestamp  string      `json:"timestamp"`
	ID         tasks.Id    `json:"id"`
	TaskStatus string      `json:"taskStatus"`
	AppID      tasks.AppId `json:"appId"`
	Version    string      `json:"version"`
	Alive      bool        `json:"alive"`
}

func ParseTaskHealthChange(event []byte) (*TaskHealthChange, error) {
	task := &TaskHealthChange{}
	err := json.Unmarshal(event, task)
	return task, err
}
