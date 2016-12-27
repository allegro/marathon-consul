package web

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/metrics"
)

type EventHandler struct {
	eventQueue   chan event
	maxEventSize int64
}

func newWebHandler(eventQueue chan event, maxEventSize int64) *EventHandler {
	if maxEventSize < 1000 {
		log.WithField("maxEventSize", maxEventSize).Warning("Max event size is too small. Switching to 1000")
		maxEventSize = 1000
	}
	return &EventHandler{
		eventQueue:   eventQueue,
		maxEventSize: maxEventSize,
	}
}

const (
	statusUpdateEventType        = "status_update_event"
	healthStatusChangedEventType = "health_status_changed_event"
)

// Handle is responsible for accepting events and passing them to event queue
// for async processing. It always returns 2xx even if requests are malformed
// to prevent marathon from suspending subscription.
// Processed events must be smaller than maxEventSize and must contain
// supported event type.
func (h *EventHandler) Handle(w http.ResponseWriter, r *http.Request) {
	metrics.Time("events.response", func() {
		limitedBody := http.MaxBytesReader(w, r.Body, h.maxEventSize)
		defer limitedBody.Close()
		body, err := ioutil.ReadAll(limitedBody)
		if err != nil {
			drop(err, w)
			return
		}

		e, err := events.ParseEvent(body)
		if err != nil {
			drop(err, w)
			return
		}

		metrics.Mark("events.requests." + e.Type)
		delay := time.Now().Unix() - e.Timestamp.Unix()
		metrics.UpdateGauge("events.requests.delay.current", delay)
		log.WithFields(log.Fields{"EventType": e.Type, "OriginalTimestamp": e.Timestamp.String()}).Debug("Received event")

		if e.Type != statusUpdateEventType && e.Type != healthStatusChangedEventType {
			drop(fmt.Errorf("%s is not supported", e.Type), w)
			return
		}

		select {
		case h.eventQueue <- event{eventType: e.Type, body: body, timestamp: time.Now()}:
			accept(w)
		default:
			metrics.Mark("events.queue.drop")
			drop(errors.New("Event queue full"), w)
		}

	})
}

func accept(w http.ResponseWriter) {
	metrics.Mark("events.response.accept")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "OK")
}

func drop(err error, w http.ResponseWriter) {
	log.WithError(err).Debug("Malformed request")
	metrics.Mark("events.response.drop")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, err.Error())
}
