package events

import (
	"encoding/json"
	"errors"
	"regexp"

	"github.com/allegro/marathon-consul/apps"
)

type TaskHealthChange struct {
	Timestamp string `json:"timestamp"`
	// Prefer TaskID() instead of ID
	ID         apps.TaskID `json:"id"`
	InstanceID string      `json:"instanceId"`
	AppID      apps.AppID  `json:"appId"`
	Version    string      `json:"version"`
	Alive      bool        `json:"alive"`
}

// Regular expression to extract runSpecId from instanceId
// See: https://github.com/mesosphere/marathon/blob/v1.4.0-RC4/src/main/scala/mesosphere/marathon/core/instance/Instance.scala#L244
var instanceIdRegex = regexp.MustCompile(`^(.+)\.(instance-|marathon-)([^\.]+)$`)

func (t TaskHealthChange) TaskID() apps.TaskID {
	if t.ID != "" {
		return t.ID
	}
	return apps.TaskID(instanceIdRegex.ReplaceAllString(t.InstanceID, "$1.$3"))
}

func ParseTaskHealthChange(event []byte) (*TaskHealthChange, error) {
	task := &TaskHealthChange{}
	err := json.Unmarshal(event, task)

	if err != nil {
		return nil, err
	}

	// Marathon 1.4 changes this event so it does not contain TaskID.
	// We need to validate this event if it contains required fields.
	// See: https://phabricator.mesosphere.com/D218#10153
	if task.ID == "" && task.InstanceID == "" {
		return nil, errors.New("Missing task ID")
	}

	return task, nil
}
