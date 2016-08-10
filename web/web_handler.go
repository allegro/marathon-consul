package web

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/metrics"
)

type MarathonEventsHandler struct {
	eventQueue chan event
}

func newWebHandler(eventQueue chan event) *MarathonEventsHandler {
	return &MarathonEventsHandler{eventQueue: eventQueue}
}

func (h *MarathonEventsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	metrics.Time("events.response", func() {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.WithError(err).Debug("Malformed request")
			handleBadRequest(err, w)
			return
		}
		log.WithField("Body", string(body)).Debug("Received request")

		eventType, err := events.EventType(body)
		if err != nil {
			handleBadRequest(err, w)
			return
		}

		log.WithField("EventType", eventType).Debug("Received event")
		metrics.Mark("events.requests." + eventType)

		h.eventQueue <- event{eventType: eventType, body: body, timestamp: time.Now()}
		accepted(w)
	})
}

func handleBadRequest(err error, w http.ResponseWriter) {
	metrics.Mark("events.response.error.400")
	w.WriteHeader(http.StatusBadRequest)
	log.WithError(err).Debug("Returning 400 due to malformed request")
	fmt.Fprintln(w, err.Error())
}

func accepted(w http.ResponseWriter) {
	metrics.Mark("events.response.202")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "OK")
}
