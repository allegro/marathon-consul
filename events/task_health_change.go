package events

import (
	"encoding/json"
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
