package marathon

import "time"

type Config struct {
	Location  string
	Protocol  string
	Username  string
	Password  string
	VerifySsl bool
	Timeout   time.Duration
}
