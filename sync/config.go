package sync

import "time"

type Config struct {
	Enabled  bool
	Force    bool
	Interval time.Duration
	Leader   string
}
