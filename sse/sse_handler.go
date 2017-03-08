package sse

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"

	log "github.com/Sirupsen/logrus"
)

// SSEHandler defines handler for marathon event stream, opening and closing
// subscription
type SSEHandler struct {
	config      Config
	eventQueue  chan events.Event
	loc         string
	client      *http.Client
	close       context.CancelFunc
	req         *http.Request
	Streamer    *marathon.Streamer
	maxLineSize int64
}

func newSSEHandler(eventQueue chan events.Event, service marathon.Marathoner, maxLineSize int64, config Config) *SSEHandler {

	streamer, err := service.EventStream(
		[]string{events.StatusUpdateEventType, events.HealthStatusChangedEventType},
		config.Retries,
		config.RetryBackoff,
	)
	if err != nil {
		log.WithError(err).Fatal("Unable to start Streamer")
	}

	return &SSEHandler{
		config:      config,
		eventQueue:  eventQueue,
		Streamer:    streamer,
		maxLineSize: maxLineSize,
	}
}

// Open connection to marathon v2/events
func (h *SSEHandler) start() chan<- events.StopEvent {
	stopChan := make(chan events.StopEvent)
	go func() {
		<-stopChan
		h.stop()
	}()

	go func() {
		defer h.stop()

		err := h.Streamer.Start()
		if err != nil {
			log.WithError(err).Error("Unable to start streamer")
		}
		// buffer used for token storage,
		// if token is greater than buffer, empty token is stored
		buffer := make([]byte, h.maxLineSize)
		// configure streamer scanner :)
		h.Streamer.Scanner.Buffer(buffer, cap(buffer))
		h.Streamer.Scanner.Split(events.ScanLines)
		for {
			metrics.Time("events.read", func() { h.handle() })
		}
	}()
	return stopChan
}

func (h *SSEHandler) handle() {
	e, err := events.ParseSSEEvent(h.Streamer.Scanner)
	if err != nil {
		if err == io.EOF {
			// Event could be partial at this point
			h.enqueueEvent(e)
		}
		log.WithError(err).Error("Error when parsing the event")
		err = h.Streamer.Recover()
		if err != nil {
			log.WithError(err).Fatalf("Unable to recover streamer")
		}
	}
	metrics.Mark("events.read." + e.Type)
	delay := time.Now().Unix() - e.Timestamp.Unix()
	metrics.UpdateGauge("events.read.delay.current", delay)
	if e.Type != events.StatusUpdateEventType && e.Type != events.HealthStatusChangedEventType {
		log.Debugf("%s is not supported", e.Type)
		metrics.Mark("events.read.drop")
		return
	}
	h.enqueueEvent(e)
}

func (h *SSEHandler) enqueueEvent(e events.SSEEvent) {
	select {
	case h.eventQueue <- events.Event{Timestamp: time.Now(), EventType: e.Type, Body: e.Body}:
		metrics.Mark("events.read.accept")
	default:
		log.Error("Events queue full. Dropping the event")
		metrics.Mark("events.read.drop")
	}
}

// Close connections managed by context
func (h *SSEHandler) stop() {
	h.Streamer.Stop()
}
