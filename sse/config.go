package sse

type Config struct {
	Enabled      bool
	Retries      int
	RetryBackoff int
}
