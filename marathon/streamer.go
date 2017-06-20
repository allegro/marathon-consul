package marathon

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"io"

	log "github.com/Sirupsen/logrus"
)

type Streamer interface {
	Stop()
	Start() (io.Reader, error)
	Recover() (io.Reader, error)
}

type streamer struct {
	scanner      *bufio.Scanner
	cancel       context.CancelFunc
	client       *http.Client
	username     string
	password     string
	subURL       string
	retries      int
	retryBackoff int
	noRecover    bool
}

func (s *streamer) Stop() {
	s.cancel()
	s.noRecover = true
}

func (s *streamer) Start() (io.Reader, error) {
	req, err := http.NewRequest("GET", s.subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to create request: %s", err)
	}
	req.SetBasicAuth(s.username, s.password)
	req.Header.Set("Accept", "text/event-stream")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	req = req.WithContext(ctx)
	res, err := s.client.Do(req)
	if err != nil {
		s.cancel()
		return nil, fmt.Errorf("Subscription request errored: %s", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Event stream not connected: Expected %d but got %d", http.StatusOK, res.StatusCode)
	}
	log.WithFields(log.Fields{
		"Host":   req.Host,
		"URI":    req.URL.RequestURI(),
		"Method": "GET",
	}).Debug("Subsciption success")
	return res.Body, nil
}

func (s *streamer) Recover() (io.Reader, error) {
	if s.noRecover {
		return nil, errors.New("Streamer is not recoverable")
	}
	s.cancel()

	reader, err := s.Start()
	for i := 0; err != nil && i <= s.retries; reader, err = s.Start() {
		seconds := time.Duration(i * s.retryBackoff)
		time.Sleep(seconds * time.Second)
		i++
	}
	if err != nil {
		return nil, fmt.Errorf("Cannot recover Streamer: %s", err)
	}
	return reader, nil
}
