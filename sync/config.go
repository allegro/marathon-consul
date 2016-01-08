package sync

import "time"

type Config struct {
	Enabled  bool
	Interval time.Duration
	Leader   string
	Force    bool
}
