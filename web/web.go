package web

import (
	"net/http"

	service "github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
)

type Stop func()
type Handler func(w http.ResponseWriter, r *http.Request)

func NewHandler(config Config, marathon marathon.Marathoner, service service.ConsulServices) (Handler, Stop) {

	stopChannels := make([]chan<- stopEvent, config.WorkersCount, config.WorkersCount)
	eventQueue := make(chan event, config.QueueSize)
	for i := 0; i < config.WorkersCount; i++ {
		handler := newEventHandler(i, service, marathon, eventQueue)
		stopChannels[i] = handler.Start()
	}
	return newWebHandler(eventQueue).Handle, stop(stopChannels)
}

func stop(channels []chan<- stopEvent) Stop {
	return func() {
		for _, channel := range channels {
			channel <- stopEvent{}
		}
	}
}
