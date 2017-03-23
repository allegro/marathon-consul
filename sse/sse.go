package sse

import (
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/service"
	"github.com/allegro/marathon-consul/web"
)

type Stop func()
type Handler func(w http.ResponseWriter, r *http.Request)

func NewHandler(config Config, webConfig web.Config, marathon marathon.Marathoner, serviceOperations service.ServiceRegistry) Stop {
	stopChannels := make([]chan<- events.StopEvent, webConfig.WorkersCount, webConfig.WorkersCount)
	eventQueue := make(chan events.Event, webConfig.QueueSize)
	for i := 0; i < webConfig.WorkersCount; i++ {
		handler := events.NewEventHandler(i, serviceOperations, marathon, eventQueue)
		stopChannels[i] = handler.Start()
	}

	sse := newSSEHandler(eventQueue, marathon, webConfig.MaxEventSize, config)
	dispatcherStop := sse.start()

	guardQuit := leaderGuard(sse.Streamer, marathon)
	stopChannels = append(stopChannels, dispatcherStop, guardQuit)

	return stop(stopChannels)
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
		ticker := time.Tick(5 * time.Second)
		for {
			select {
			case <-ticker:
				if iAMLeader, err := m.IsLeader(); !iAMLeader && err != nil {
					// Leader changed, not revocerable.
					s.Stop()
					log.Error("Tearing down SSE stream, marathon leader changed.")
					return
				} else if err != nil {
					log.WithError(err).Error("Leader Guard error while checking leader.")
				}
			case <-quit:
				log.Info("Recieved quit notification. Quit checker")
				s.Stop()
				return
			}
		}
	}()
	return quit
}
