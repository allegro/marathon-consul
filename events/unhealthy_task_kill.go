package events

import (
	"encoding/json"

	"github.com/allegro/marathon-consul/apps"
)

type UnhealthyTaskKilled struct {
	Timestamp string      `json:"timestamp"`
	ID        apps.TaskID `json:"taskId"`
	AppID     apps.AppID  `json:"appId"`
	Version   string      `json:"version"`
}

func ParseUnhealthyTaskKilled(event []byte) (*UnhealthyTaskKilled, error) {
	task := &UnhealthyTaskKilled{}
	err := json.Unmarshal(event, task)
	return task, err
}
