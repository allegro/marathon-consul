package marathon

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Streamer struct {
	Scanner      *bufio.Scanner
	cancel       context.CancelFunc
	client       *http.Client
	subURL       string
	retries      int
	retryBackoff int
	noRecover    bool
}

func (s *Streamer) stop() {
	s.cancel()
}
func (s *Streamer) Stop() {
	s.noRecover = true
	s.stop()
}

func (s *Streamer) Start() error {
	req, err := http.NewRequest("GET", s.subURL, nil)
	if err != nil {
		log.Fatal("Unable to create Streamer request")
		return nil
	}
	req.Header.Set("Accept", "text/event-stream")
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	req = req.WithContext(ctx)
	res, err := s.client.Do(req)
	if err != nil {
		s.cancel()
		return err
	}
	if res.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"Location": s.subURL,
			"Method":   "GET",
		}).Errorf("Got status code : %d", res.StatusCode)
	}
	log.WithFields(log.Fields{
		"SubUrl": s.subURL,
		"Method": "GET",
	}).Debug("Subsciption success")
	s.Scanner = bufio.NewScanner(res.Body)

	return nil
}

func (s *Streamer) Recover() error {
	if s.noRecover {
		return fmt.Errorf("Streamer is not recoverable")
	}
	s.stop()

	err := s.Start()
	i := 0
	for ; err != nil && i <= s.retries; err = s.Start() {
		seconds := time.Duration(i * s.retryBackoff)
		time.Sleep(seconds * time.Second)
		i++
	}
	return err
}
