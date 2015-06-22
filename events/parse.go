package events

import (
	"encoding/json"
	"errors"
)

var (
	ErrNoEvent = errors.New("no event")
)

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

// ParseAPIPostEvent parses api_post_event
func parseAPIPostEvent(jsonBlob []byte) (Event, error) {
	event := APIPostEvent{}
	err := json.Unmarshal(jsonBlob, &event)
	return event, err
}

// ParseDeploymentInfoEvent parses deployment_info
func parseDeploymentInfoEvent(jsonBlob []byte) (Event, error) {
	event := DeploymentInfoEvent{}
	err := json.Unmarshal(jsonBlob, &event)
	return event, err
}

// ParseAppTerminatedEvent parses app_terminated_event
func parseAppTerminatedEvent(jsonBlob []byte) (Event, error) {
	event := AppTerminatedEvent{}
	err := json.Unmarshal(jsonBlob, &event)
	return event, err
}

// ParseEvent combines the functions in this module to return an event without
// the user having to worry about the *type* of the event.
func ParseEvent(jsonBlob []byte) (event Event, err error) {
	eventType, err := EventType(jsonBlob)
	if err != nil {
		return
	}

	switch eventType {
	case "api_post_event":
		event, err = parseAPIPostEvent(jsonBlob)
	case "deployment_info":
		event, err = parseDeploymentInfoEvent(jsonBlob)
	case "app_terminated_event":
		event, err = parseAppTerminatedEvent(jsonBlob)
	}

	return
}
