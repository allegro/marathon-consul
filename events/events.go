package events

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Timestamp struct {
	time.Time
}

func (t *Timestamp) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		t.Time = time.Time{}
		return
	}
	t.Time, err = time.Parse(time.RFC3339Nano, s)
	return
}

func (t *Timestamp) String() string {
	return t.Format(time.RFC3339Nano)
}

type Event struct {
	Type      string    `json:"eventType"`
	Timestamp Timestamp `json:"timestamp"`
}

func ParseEvent(jsonBlob []byte) (Event, error) {
	event := Event{}
	err := json.Unmarshal(jsonBlob, &event)
	if err != nil {
		return Event{}, err
	} else if event.Type == "" {
		return Event{}, errors.New("Missing event type")
	} else if event.Timestamp.Unix() == (time.Time{}).Unix() {
		return Event{}, errors.New("Missing timestamp")
	}
	return event, nil
}
