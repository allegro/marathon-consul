package events

import (
	"encoding/json"
	"errors"

	"github.com/allegro/marathon-consul/apps"
)

type Event interface {
	Apps() []*apps.App
	GetType() string
}

type baseEvent struct {
	Type string `json:"eventType"`
}

func EventType(jsonBlob []byte) (string, error) {
	event := baseEvent{}
	err := json.Unmarshal(jsonBlob, &event)
	if err != nil {
		return "", err
	} else if event.Type == "" {
		return "", errors.New("no event")
	} else {
		return event.Type, nil
	}
}
