package marathon

import "github.com/allegro/marathon-consul/time"

type Config struct {
	Location  string
	Protocol  string
	Username  string
	Password  string
	Leader    string
	VerifySsl bool
	Timeout   time.Interval
}
