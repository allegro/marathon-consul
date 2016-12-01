package time

import (
	"encoding/json"
	"strconv"
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

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Timestamp) String() string {
	return t.Format(time.RFC3339Nano)
}

type Interval struct {
	time.Duration
}

func (t *Interval) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		t.Duration = 0
		return nil
	} else if value, e := strconv.ParseInt(s, 10, 64); e == nil {
		t.Duration = time.Duration(value)
		return nil
	}
	t.Duration, err = time.ParseDuration(s)
	return err
}

func (t Interval) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Duration.String())
}
