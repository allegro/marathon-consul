package web

import (
	"net/http"

	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/service"
)

type Stop func()
type Handler func(w http.ResponseWriter, r *http.Request)

func NewHandler(config Config, marathon marathon.Marathoner, serviceOperations service.ServiceRegistry) (Handler, Stop) {

	stopChannels := make([]chan<- events.StopEvent, config.WorkersCount, config.WorkersCount)
	eventQueue := make(chan events.Event, config.QueueSize)
	for i := 0; i < config.WorkersCount; i++ {
		handler := events.NewEventHandler(i, serviceOperations, marathon, eventQueue)
		stopChannels[i] = handler.Start()
	}
	return newWebHandler(eventQueue, config.MaxEventSize).Handle, stop(stopChannels)
}

func stop(channels []chan<- events.StopEvent) Stop {
	return func() {
		for _, channel := range channels {
			channel <- events.StopEvent{}
		}
	}
}
