package web

import (
	"net/http"

	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/service"
)

type Stop func()
type Handler func(w http.ResponseWriter, r *http.Request)

func NewHandler(config Config, marathon marathon.Marathoner, serviceOperations service.ServiceRegistry) (Handler, Stop) {

	stopChannels := make([]chan<- stopEvent, config.WorkersCount, config.WorkersCount)
	eventQueue := make(chan event, config.QueueSize)
	for i := 0; i < config.WorkersCount; i++ {
		handler := newEventHandler(i, serviceOperations, marathon, eventQueue)
		stopChannels[i] = handler.start()
	}
	return newWebHandler(eventQueue, config.MaxEventSize).Handle, stop(stopChannels)
}

func stop(channels []chan<- stopEvent) Stop {
	return func() {
		for _, channel := range channels {
			channel <- stopEvent{}
		}
	}
}
