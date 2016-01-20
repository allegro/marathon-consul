package events

import (
	"encoding/json"
	"errors"
	"github.com/allegro/marathon-consul/apps"
)

var (
	ErrNoEvent = errors.New("no event")
)

type Event interface {
	Apps() []*apps.App
	GetType() string
}

type BaseEvent struct {
	Type string `json:"eventType"`
}

func EventType(jsonBlob []byte) (string, error) {
	event := BaseEvent{}
	err := json.Unmarshal(jsonBlob, &event)
	if err != nil {
		return "", err
	} else if event.Type == "" {
		return "", ErrNoEvent
	} else {
		return event.Type, nil
	}
}
