package marathon

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type Streamer struct {
	Scanner      *bufio.Scanner
	cancel       context.CancelFunc
	client       *http.Client
	username     string
	password     string
	subURL       string
	retries      int
	retryBackoff time.Duration
	noRecover    bool
}

func (s *Streamer) Stop() {
	s.cancel()
	s.noRecover = true
}

func (s *Streamer) Start() error {
	req, err := http.NewRequest("GET", s.subURL, nil)
	if err != nil {
		return fmt.Errorf("Unable to create request: %s", err)
	}
	req.SetBasicAuth(s.username, s.password)
	req.Header.Set("Accept", "text/event-stream")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	req = req.WithContext(ctx)
	res, err := s.client.Do(req)
	if err != nil {
		s.cancel()
		return fmt.Errorf("Subscription request errored: %s", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Event stream not connected: Expected %d but got %d", http.StatusOK, res.StatusCode)
	}
	log.WithFields(log.Fields{
		"Host":   req.Host,
		"URI":    req.URL.RequestURI(),
		"Method": "GET",
	}).Debug("Subsciption success")
	s.Scanner = bufio.NewScanner(res.Body)

	return nil
}

func (s *Streamer) Recover() error {
	if s.noRecover {
		return errors.New("Streamer is not recoverable")
	}
	s.cancel()

	err := s.Start()
	i := 0
	for ; err != nil && i <= s.retries; err = s.Start() {
		time.Sleep(s.retryBackoff)
		i++
	}
	if err != nil {
		return fmt.Errorf("Cannot recover Streamer: %s", err)
	}
	return nil
}
