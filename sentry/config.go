package sentry

import "github.com/allegro/marathon-consul/time"

type Config struct {
	DSN     string
	Env     string
	Level   string
	Release string
	Timeout time.Interval
}
