package metrics

import "time"

type Config struct {
	Target   string
	Prefix   string
	Interval time.Duration
	Addr     string
}
