package sse

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/service"
	"github.com/allegro/marathon-consul/web"
)

type Stop func()
type Handler func(w http.ResponseWriter, r *http.Request)

func NewHandler(config Config, webConfig web.Config, marathon marathon.Marathoner, serviceOperations service.Registry) (Stop, error) {
	stopChannels := make([]chan<- events.StopEvent, webConfig.WorkersCount)
	stopFunc := stop(stopChannels)
	eventQueue := make(chan events.Event, webConfig.QueueSize)
	for i := 0; i < webConfig.WorkersCount; i++ {
		handler := events.NewEventHandler(i, serviceOperations, marathon, eventQueue)
		stopChannels[i] = handler.Start()
	}

	sse, err := newSSEHandler(eventQueue, marathon, webConfig.MaxEventSize, config)
	if err != nil {
		stopFunc()
		return nil, fmt.Errorf("Cannot create SSE handler: %s", err)
	}
	dispatcherStop, err := sse.start()
	if err != nil {
		stopFunc()
		return nil, fmt.Errorf("Cannot start SSE handler: %s", err)
	}

	guardQuit := leaderGuard(sse.Streamer, marathon)
	stopChannels = append(stopChannels, dispatcherStop, guardQuit)

	return stop(stopChannels), nil
}

func stop(channels []chan<- events.StopEvent) Stop {
	return func() {
		for _, channel := range channels {
			channel <- events.StopEvent{}
		}
	}
}

// leaderGuard is a watchdog goroutine,
// periodically checks if leader has changed
// if change is detected, passed streamer is stopped - unable to recover
// if this goroutine is quited, agent is stopped - unable to recover
func leaderGuard(s *marathon.Streamer, m marathon.Marathoner) chan<- events.StopEvent {
	// TODO(tz) - consider launching this goroutine from marathon,
	// no need to pass marathon reciever then ??
	quit := make(chan events.StopEvent)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				if iAMLeader, err := m.IsLeader(); !iAMLeader && err != nil {
					// Leader changed, not revocerable.
					ticker.Stop()
					s.Stop()
					log.Error("Tearing down SSE stream, marathon leader changed.")
					return
				} else if err != nil {
					log.WithError(err).Error("Leader Guard error while checking leader.")
				}
			case <-quit:
				log.Info("Recieved quit notification. Quit checker")
				ticker.Stop()
				s.Stop()
				return
			}
		}
	}()
	return quit
}
