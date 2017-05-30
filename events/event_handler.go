package events

import (
	"bytes"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/service"
)

type Event struct {
	Timestamp time.Time
	EventType string
	Body      []byte
}

type eventHandler struct {
	id              int
	serviceRegistry service.ServiceRegistry
	marathon        marathon.Marathoner
	eventQueue      <-chan Event
}

type StopEvent struct{}

const (
	StatusUpdateEventType        = "status_update_event"
	HealthStatusChangedEventType = "health_status_changed_event"
)

func NewEventHandler(id int, serviceRegistry service.ServiceRegistry, marathon marathon.Marathoner, eventQueue <-chan Event) *eventHandler {
	return &eventHandler{
		id:              id,
		serviceRegistry: serviceRegistry,
		marathon:        marathon,
		eventQueue:      eventQueue,
	}
}

func (fh *eventHandler) Start() chan<- StopEvent {
	var e Event
	process := func() {
		err := fh.handleEvent(e.EventType, e.Body)
		if err != nil {
			metrics.Mark("events.processing.error")
		} else {
			metrics.Mark("events.processing.succes")
		}
	}

	quitChan := make(chan StopEvent)
	log.WithField("Id", fh.id).Println("Starting worker")
	go func() {
		for {
			select {
			case e = <-fh.eventQueue:
				metrics.Mark(fmt.Sprintf("events.handler.%d", fh.id))

				queueLength := int64(len(fh.eventQueue))
				metrics.UpdateGauge("events.queue.len", queueLength)
				queueCapacity := int64(cap(fh.eventQueue))

				utilization := int64(0)
				if queueCapacity > 0 {
					utilization = 100 * (queueLength / queueCapacity)
				}
				metrics.UpdateGauge("events.queue.util", utilization)

				metrics.UpdateGauge("events.queue.delay_ns", time.Since(e.Timestamp).Nanoseconds())
				metrics.Time("events.processing."+e.EventType, process)
			case <-quitChan:
				log.WithField("Id", fh.id).Info("Stopping worker")
			}
		}
	}()
	return quitChan
}

func (fh *eventHandler) handleEvent(eventType string, body []byte) error {

	body = replaceTaskIDWithID(body)

	switch eventType {
	case StatusUpdateEventType:
		return fh.handleStatusEvent(body)
	case HealthStatusChangedEventType:
		return fh.handleHealthyTask(body)
	default:
		err := fmt.Errorf("Unsuported event type: %s", eventType)
		log.WithError(err).WithField("EventType", eventType).Error("This should never happen. Not handled event type")
		return err
	}
}

func (fh *eventHandler) handleHealthyTask(body []byte) error {
	taskHealthChange, err := ParseTaskHealthChange(body)
	if err != nil {
		log.WithError(err).Error("Body generated error")
		return err
	}
	delay := taskHealthChange.Timestamp.Delay()
	metrics.UpdateGauge("events.read.delay.current", int64(delay))

	appID := taskHealthChange.AppID
	taskID := taskHealthChange.TaskID()
	log.WithFields(log.Fields{"taskID": taskID, "appId": appID}).Info("Got HealthStatusEvent")

	if !taskHealthChange.Alive {
		log.WithField("Id", taskID).Debug("Task is not alive. Not registering")
		return nil
	}

	app, err := fh.marathon.App(appID)
	if err != nil {
		log.WithField("Id", taskID).WithError(err).Error("There was a problem obtaining app info")
		return err
	}

	if !app.IsConsulApp() {
		err = fmt.Errorf("%s is not consul app. Missing consul label", app.ID)
		log.WithField("Id", taskID).WithError(err).Debug("Skipping app registration in Consul")
		return nil
	}

	tasks := app.Tasks

	task, found := apps.FindTaskByID(taskID, tasks)
	if !found {
		log.WithField("Id", taskID).Error("Task not found")
		return err
	}

	if task.IsHealthy() {
		err := fh.serviceRegistry.Register(&task, app)
		if err != nil {
			log.WithField("Id", task.ID).WithError(err).Error("There was a problem registering task")
			return err
		}
		return nil
	}
	log.WithField("Id", task.ID).Debug("Task is not healthy. Not registering")
	return nil
}

func (fh *eventHandler) handleStatusEvent(body []byte) error {
	task, err := ParseTaskHealthChange(body)
	if err != nil {
		log.WithError(err).WithField("Body", string(body[:])).Error("Could not parse event body")
		return err
	}
	delay := task.Timestamp.Delay()
	metrics.UpdateGauge("events.read.delay.current", int64(delay))

	appID := task.AppID
	taskID := task.TaskID()

	log.WithFields(log.Fields{
		"taskId":     taskID,
		"appID":      appID,
		"TaskStatus": task.TaskStatus,
	}).Info("Got StatusEvent")

	switch task.TaskStatus {
	case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLING", "TASK_KILLED", "TASK_LOST":
		app, err := fh.marathon.App(appID)
		if err != nil {
			log.WithField("Id", taskID).WithError(err).Error("There was a problem obtaining app info")
			return err
		}

		if !app.IsConsulApp() {
			err = fmt.Errorf("%s is not consul app. Missing consul label", app.ID)
			log.WithField("Id", taskID).WithError(err).Debug("Skipping app registration in Consul")
			return nil
		}
		return fh.deregister(task.ID)
	default:
		log.WithFields(log.Fields{
			"Id":         task.ID,
			"taskStatus": task.TaskStatus,
		}).Debug("Not handled task status")
		return nil
	}
}

func (fh *eventHandler) deregister(taskID apps.TaskID) error {
	err := fh.serviceRegistry.DeregisterByTask(taskID)
	if err != nil {
		log.WithField("Id", taskID).WithError(err).Error("There was a problem deregistering task")
	}
	return err
}

// for every other use of Tasks, Marathon uses the "id" field for the task ID.
// Here, it uses "taskId", with most of the other fields being equal. We'll
// just swap "taskId" for "id" in the body so that we can successfully parse
// incoming events.
func replaceTaskIDWithID(body []byte) []byte {
	return bytes.Replace(body, []byte("taskId"), []byte("id"), -1)
}
