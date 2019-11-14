package apps

import (
	"encoding/json"
	"strings"

	"github.com/allegro/marathon-consul/time"
)

type Task struct {
	ID TaskID `json:"id"`
	// Timestamp field is not a part of a Marathon task object.
	// It's only present in StatusUpdateEventType and we are using this struct for decoding it.
	// As well as for Marathon Task.
	Timestamp          time.Timestamp      `json:"timestamp"`
	TaskStatus         string              `json:"taskStatus"`
	State              string              `json:"state"`
	AppID              AppID               `json:"appId"`
	Host               string              `json:"host"`
	Ports              []int               `json:"ports"`
	HealthCheckResults []HealthCheckResult `json:"healthCheckResults"`
}

// Marathon Task ID
// Usually in the form of AppId.uuid with '/' replaced with '_'
type TaskID string

func (id TaskID) String() string {
	return string(id)
}

func (id TaskID) AppID() AppID {
	index := strings.LastIndex(id.String(), ".")
	return AppID("/" + strings.Replace(id.String()[0:index], "_", "/", -1))
}

type HealthCheckResult struct {
	Alive bool `json:"alive"`
}

type TasksResponse struct {
	Tasks []Task `json:"tasks"`
}

func FindTaskByID(id TaskID, tasks []Task) (Task, bool) {
	for _, task := range tasks {
		if strings.HasPrefix(task.ID.String(), id.String()) && (len(id.String()) > 36) {
			return task, true
		}
	}
	return Task{}, false
}

func ParseTasks(jsonBlob []byte) ([]Task, error) {
	tasks := &TasksResponse{}
	err := json.Unmarshal(jsonBlob, tasks)

	return tasks.Tasks, err
}

func ParseTask(event []byte) (*Task, error) {
	task := &Task{}
	err := json.Unmarshal(event, task)
	return task, err
}

func (t Task) IsHealthy() bool {
	if len(t.HealthCheckResults) < 1 {
		return false
	}
	register := true
	for _, healthCheckResult := range t.HealthCheckResults {
		register = register && healthCheckResult.Alive && t.State != "TASK_KILLING"
	}
	return register
}
