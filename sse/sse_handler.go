package sse

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/metrics"
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
	streamer    marathon.Streamer
	maxLineSize int64
	scanner     *bufio.Scanner
}

func newSSEHandler(eventQueue chan events.Event, service marathon.Marathoner, maxLineSize int64, config Config) (*SSEHandler, error) {

	streamer, err := service.EventStream(
		[]string{events.StatusUpdateEventType, events.HealthStatusChangedEventType},
		config.Retries,
		config.RetryBackoff,
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to start Streamer: %s", err)
	}

	return &SSEHandler{
		config:      config,
		eventQueue:  eventQueue,
		streamer:    streamer,
		maxLineSize: maxLineSize,
	}, nil
}

// Open connection to marathon v2/events
func (h *SSEHandler) start() (chan<- events.StopEvent, error) {
	reader, err := h.streamer.Start()
	if err != nil {
		return nil, fmt.Errorf("Cannot start Streamer: %s", err)
	}

	h.newScanner(reader)

	stopChan := make(chan events.StopEvent)
	go func() {
		<-stopChan
		h.stop()
	}()

	go func() {
		defer h.stop()
		for {
			metrics.Time("events.read", func() { h.handle() })
		}
	}()
	return stopChan, nil
}

func (h *SSEHandler) handle() {
	e, err := events.ParseSSEEvent(h.scanner)
	if err != nil {
		if err == io.EOF {
			// Event could be partial at this point
			h.enqueueEvent(e)
		}
		log.WithError(err).Error("Error when parsing the event")
		reader, err := h.streamer.Recover()
		if err != nil {
			log.WithError(err).Fatalf("Unable to recover streamer")
		}
		h.newScanner(reader)
	}
	metrics.Mark("events.read." + e.Type)
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

func (h *SSEHandler) newScanner(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	// buffer used for token storage,
	// if token is greater than buffer, empty token is stored
	buffer := make([]byte, h.maxLineSize)
	scanner.Buffer(buffer, cap(buffer))
	scanner.Split(events.ScanLines)
	h.scanner = scanner
}

// Close connections managed by context
func (h *SSEHandler) stop() {
	h.streamer.Stop()
}
