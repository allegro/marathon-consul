package metrics

import "github.com/allegro/marathon-consul/time"

type Config struct {
	Target   string
	Prefix   string
	Interval time.Interval
	Addr     string
}
