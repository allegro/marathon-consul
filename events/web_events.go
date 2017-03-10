package events

import (
	"encoding/json"
	"errors"

	"github.com/allegro/marathon-consul/time"
)

type WebEvent struct {
	Type      string         `json:"eventType"`
	Timestamp time.Timestamp `json:"timestamp"`
}

func ParseEvent(jsonBlob []byte) (WebEvent, error) {
	webEvent := WebEvent{}
	err := json.Unmarshal(jsonBlob, &webEvent)
	if err != nil {
		return WebEvent{}, err
	} else if webEvent.Type == "" {
		return WebEvent{}, errors.New("Missing event type")
	} else if webEvent.Timestamp.Missing() {
		return WebEvent{}, errors.New("Missing timestamp")
	}
	return webEvent, nil
}
