package web

type Config struct {
	Listen       string
	QueueSize    int
	WorkersCount int
	MaxEventSize int64
}
