package sse

import "github.com/allegro/marathon-consul/time"

type Config struct {
	Retries      int
	RetryBackoff time.Interval
}
