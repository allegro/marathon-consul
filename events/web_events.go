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

type WebEvent struct {
	Type      string    `json:"eventType"`
	Timestamp Timestamp `json:"timestamp"`
}

func ParseEvent(jsonBlob []byte) (WebEvent, error) {
	webEvent := WebEvent{}
	err := json.Unmarshal(jsonBlob, &webEvent)
	if err != nil {
		return WebEvent{}, err
	} else if webEvent.Type == "" {
		return WebEvent{}, errors.New("Missing event type")
	} else if webEvent.Timestamp.Unix() == (time.Time{}).Unix() {
		return WebEvent{}, errors.New("Missing timestamp")
	}
	return webEvent, nil
}
