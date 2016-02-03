package web

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	service "github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/events"
	marathon "github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
	"io/ioutil"
	"net/http"
)

type EventHandler struct {
	service  service.ConsulServices
	marathon marathon.Marathoner
}

func NewEventHandler(service service.ConsulServices, marathon marathon.Marathoner) *EventHandler {
	return &EventHandler{
		service:  service,
		marathon: marathon,
	}
}

func (fh *EventHandler) Handle(w http.ResponseWriter, r *http.Request) {
	metrics.Time("events.response", func() { fh.handle(w, r) })
}

func (fh *EventHandler) handle(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Debug("Malformed request")
		fh.handleBadRequest(err, w)
		return
	}
	log.WithField("Body", string(body)).Debug("Received request")

	eventType, err := events.EventType(body)
	if err != nil {
		fh.handleBadRequest(err, w)
		return
	}

	fh.markEventRequest(eventType)

	log.WithField("EventType", eventType).Debug("Received event")

	switch eventType {
	case "status_update_event":
		fh.handleStatusEvent(w, body)
	case "health_status_changed_event":
		fh.handleHealthStatusEvent(w, body)
	case "deployment_info":
		fh.handleDeploymentInfo(w, body)
	case "deployment_step_success":
		fh.handleDeploymentStepSuccess(w, body)
	default:
		log.WithField("EventType", eventType).Debug("Not handled event type")
		fh.okResponse(w)
	}

	fh.markSuccess()
}

func (fh *EventHandler) handleHealthStatusEvent(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	taskHealthChange, err := events.ParseTaskHealthChange(body)
	if err != nil {
		log.WithError(err).Error("Body generated error")
		fh.handleBadRequest(err, w)
		return
	}

	log.WithFields(
		log.Fields{
			"Id":         taskHealthChange.ID,
			"TaskStatus": taskHealthChange.TaskStatus,
		}).Info("Got HealthStatusEvent")

	if !taskHealthChange.Alive {
		err := fmt.Errorf("Task %s is not healthy. Not registering", taskHealthChange.ID)
		log.WithField("Id", taskHealthChange.ID).WithError(err).Debug("Task is not healthy. Not registering")
		fh.okResponse(w)
		return
	}

	appId := taskHealthChange.AppID
	app, err := fh.marathon.App(appId)
	if err != nil {
		log.WithField("Id", taskHealthChange.ID).WithError(err).Error("There was a problem obtaining app info")
		fh.handleError(err, w)
		return
	}

	if !app.IsConsulApp() {
		err = fmt.Errorf("%s is not consul app. Missing consul label", app.ID)
		log.WithField("Id", taskHealthChange.ID).WithError(err).Debug("Skipping app registration in Consul")
		fh.okResponse(w)
		return
	}

	tasks := app.Tasks

	task, err := findTaskById(taskHealthChange.ID, tasks)
	if err != nil {
		log.WithField("Id", taskHealthChange.ID).WithError(err).Error("Task not found")
		fh.handleError(err, w)
		return
	}

	if task.IsHealthy() {
		err := fh.service.Register(&task, app)
		if err != nil {
			log.WithField("Id", task.ID).WithError(err).Error("There was a problem registering task")
			fh.handleError(err, w)
		} else {
			fh.okResponse(w)
		}
	} else {
		err := fmt.Errorf("Task %s is not healthy. Not registering", task.ID)
		log.WithField("Id", task.ID).WithError(err).Debug("Task is not healthy. Not registering")
		fh.okResponse(w)
	}
}

func findTaskById(id apps.TaskId, tasks_ []apps.Task) (apps.Task, error) {
	for _, task := range tasks_ {
		if task.ID == id {
			return task, nil
		}
	}
	return apps.Task{}, fmt.Errorf("Task %s not found", id)
}

func (fh *EventHandler) handleStatusEvent(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	task, err := apps.ParseTask(body)

	if err != nil {
		log.WithError(err).WithField("Body", body).Error("Could not parse event body")
		fh.handleBadRequest(err, w)
		return
	}

	log.WithFields(log.Fields{
		"Id":         task.ID,
		"TaskStatus": task.TaskStatus,
	}).Info("Got StatusEvent")

	switch task.TaskStatus {
	case "TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST":
		app, err := fh.marathon.App(task.AppID)
		if err != nil {
			log.WithField("Id", task.AppID).WithError(err).Error("There was a problem obtaining app info")
			fh.handleError(err, w)
			return
		}
		serviceName := app.ConsulServiceName()
		services, err := fh.service.GetServices(serviceName)
		if err != nil {
			log.WithField("AppId", app.ID).WithField("ServiceName", serviceName).WithError(err).Error("There was a problem getting Consul services")
			fh.handleError(err, w)
			return
		}

		if len(services) == 0 {
			log.WithField("AppId", app.ID).WithField("ServiceName", serviceName).Info("No matching Consul services found")
			fh.okResponse(w)
			return
		}

		for _, service := range services {
			if service.ServiceID == task.ID.String() {
				err = fh.service.Deregister(apps.TaskId(service.ServiceID), service.Address)
				if err != nil {
					log.WithField("Id", service.ServiceID).WithError(err).Error("There was a problem deregistering task")
				}
			}
		}
	default:
		log.WithFields(log.Fields{
			"Id":         task.ID,
			"taskStatus": task.TaskStatus,
		}).Debug("Not handled task status")
	}
	fh.okResponse(w)
}

/*
	This handler is used when an application is stopped
*/
func (fh *EventHandler) handleDeploymentInfo(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	deploymentEvent, err := events.ParseDeploymentEvent(body)

	if err != nil {
		log.WithError(err).WithField("Body", body).Error("Could not parse event body")
		fh.handleBadRequest(err, w)
		return
	}

	errors := []error{}
	for _, app := range deploymentEvent.StoppedConsulApps() {
		for _, error := range fh.deregisterAllAppServices(app) {
			errors = append(errors, error)
		}
	}
	if len(errors) > 0 {
		fh.handleError(fh.mergeDeregistrationErrors(errors), w)
		return
	}
	fh.okResponse(w)
}

/*
	This handler is used when an application is restarted and renamed
*/
func (fh *EventHandler) handleDeploymentStepSuccess(w http.ResponseWriter, body []byte) {
	body = replaceTaskIdWithId(body)
	deploymentEvent, err := events.ParseDeploymentEvent(body)

	if err != nil {
		log.WithError(err).WithField("Body", body).Error("Could not parse event body")
		fh.handleBadRequest(err, w)
		return
	}

	errors := []error{}
	for _, app := range deploymentEvent.RenamedConsulApps() {
		for _, error := range fh.deregisterAllAppServices(app) {
			errors = append(errors, error)
		}
	}
	if len(errors) > 0 {
		fh.handleError(fh.mergeDeregistrationErrors(errors), w)
		return
	}
	fh.okResponse(w)
}

func (fh *EventHandler) deregisterAllAppServices(app *apps.App) []error {

	errors := []error{}
	serviceName := app.ConsulServiceName()

	log.WithField("AppId", app.ID).WithField("ServiceName", serviceName).Info("Deregistering all services")

	services, err := fh.service.GetServices(serviceName)

	if err != nil {
		log.WithField("Id", app.ID).WithError(err).Error("There was a problem getting Consul services")
		errors = append(errors, err)
		return errors
	}

	if len(services) == 0 {
		log.WithField("AppId", app.ID).WithField("ServiceName", serviceName).Info("No matching Consul services found")
		return errors
	}

	for _, service := range services {
		err = fh.service.Deregister(apps.TaskId(service.ServiceID), service.Address)
		if err != nil {
			log.WithField("Id", service.ServiceID).WithError(err).Error("There was a problem deregistering task")
			errors = append(errors, err)
		}
	}
	return errors
}

func (fh *EventHandler) mergeDeregistrationErrors(errors []error) error {
	errMessage := fmt.Sprintf("%d errors occured handling service deregistration", len(errors))
	for i, err := range errors {
		errMessage = fmt.Sprintf("%s\n%d: %s", errMessage, i, err.Error())
	}
	return fmt.Errorf(errMessage)
}

func replaceTaskIdWithId(body []byte) []byte {
	// for every other use of Tasks, Marathon uses the "id" field for the task ID.
	// Here, it uses "taskId", with most of the other fields being equal. We'll
	// just swap "taskId" for "id" in the body so that we can successfully parse
	// incoming events.
	return bytes.Replace(body, []byte("taskId"), []byte("id"), -1)
}

func (fh *EventHandler) markEventRequest(event string) {
	metrics.Mark("events.requests." + event)
}

func (fh *EventHandler) markSuccess() {
	metrics.Mark("events.response.success")
}

func (fh *EventHandler) handleError(err error, w http.ResponseWriter) {
	metrics.Mark("events.response.error.500")
	w.WriteHeader(500)
	log.WithError(err).Debug("Returning 500 due to error")
	fmt.Fprintln(w, err.Error())
}

func (fh *EventHandler) handleBadRequest(err error, w http.ResponseWriter) {
	metrics.Mark("events.response.error.400")
	w.WriteHeader(400)
	log.WithError(err).Debug("Returning 400 due to malformed request")
	fmt.Fprintln(w, err.Error())
}

func (fh *EventHandler) okResponse(w http.ResponseWriter) {
	w.WriteHeader(200)
	fmt.Fprintln(w, "OK")
}
