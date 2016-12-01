package sync

import "github.com/allegro/marathon-consul/time"

type Config struct {
	Enabled  bool
	Force    bool
	Interval time.Interval
	Leader   string
}
