package marathon

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/metrics"
)

type Marathoner interface {
	ConsulApps() ([]*apps.App, error)
	App(apps.AppID) (*apps.App, error)
	Tasks(apps.AppID) ([]apps.Task, error)
	Leader() (string, error)
	EventStream([]string, int, int) (*Streamer, error)
	AmILeader() (bool, error)
}

type Marathon struct {
	Location string
	Protocol string
	MyLeader string
	Auth     *url.Userinfo
	client   *http.Client
}

type LeaderResponse struct {
	Leader string `json:"leader"`
}

func New(config Config, leader string) (*Marathon, error) {
	var auth *url.Userinfo
	if len(config.Username) == 0 && len(config.Password) == 0 {
		auth = nil
	} else {
		auth = url.UserPassword(config.Username, config.Password)
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.VerifySsl,
		},
	}
	// TODO(tz) - consider passing desiredEvents as config
	return &Marathon{
		Location: config.Location,
		Protocol: config.Protocol,
		Auth:     auth,
		MyLeader: leader,
		client: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout.Duration,
		},
	}, nil
}

func (m Marathon) App(appID apps.AppID) (*apps.App, error) {
	log.WithField("Location", m.Location).Debug("Asking Marathon for " + appID)

	body, err := m.get(m.urlWithQuery(fmt.Sprintf("/v2/apps/%s", appID), params{"embed": []string{"apps.tasks"}}))
	if err != nil {
		return nil, err
	}

	return apps.ParseApp(body)
}

func (m Marathon) ConsulApps() ([]*apps.App, error) {
	log.WithField("Location", m.Location).Debug("Asking Marathon for apps")
	body, err := m.get(m.urlWithQuery("/v2/apps", params{"embed": []string{"apps.tasks"}, "label": []string{apps.MarathonConsulLabel}}))
	if err != nil {
		return nil, err
	}

	return apps.ParseApps(body)
}

func (m Marathon) Tasks(app apps.AppID) ([]apps.Task, error) {
	log.WithFields(log.Fields{
		"Location": m.Location,
		"Id":       app,
	}).Debug("asking Marathon for tasks")

	trimmedAppID := strings.Trim(app.String(), "/")
	body, err := m.get(m.url(fmt.Sprintf("/v2/apps/%s/tasks", trimmedAppID)))
	if err != nil {
		return nil, err
	}

	return apps.ParseTasks(body)
}

func (m Marathon) Leader() (string, error) {
	log.WithField("Location", m.Location).Debug("Asking Marathon for leader")

	body, err := m.get(m.url("/v2/leader"))
	if err != nil {
		return "", err
	}

	leaderResponse := &LeaderResponse{}
	err = json.Unmarshal(body, leaderResponse)

	return leaderResponse.Leader, err
}

// EventStream method creates Streamer handler which is configured based on marathon
// client and credentials.
func (m Marathon) EventStream(desiredEvents []string, retries, retryBackoff int) (*Streamer, error) {
	subURL := m.urlWithQuery("/v2/events", params{"event_type": desiredEvents})

	// Before creating actual streamer, this function blocks until configured leader for this (m) reciever is elected.
	// When leaderPoll function successfully exit this instance of marathon-consul,
	// consider itself as a new leader and initializes Streamer.
	err := m.leaderPoll()
	if err != nil {
		log.WithError(err).Fatal("Leader poll failed. Check marathon and previous errors. Exiting")
	}

	return &Streamer{
		subURL: subURL,
		client: &http.Client{
			Transport: m.client.Transport,
		},
		retries:      retries,
		retryBackoff: retryBackoff,
	}, nil
}

// leaderPoll just blocks until configured myleader is equal to
// leader returned from marathon (/v2/leader endpoint)
func (m Marathon) leaderPoll() error {
	pollTicker := time.Tick(1 * time.Second)
	retries := 5
	i := 0
	for range pollTicker {
		leading, err := m.AmILeader()
		if err != nil {
			if i >= retries {
				return fmt.Errorf("Failed to get a leader after %d retries", i)
			}
			i++
			log.WithError(err).Error("Error while getting leader")
			continue
		}
		if leading {
			return nil
		}
		log.Debug("I am not leader")
	}
	return nil
}

func (m Marathon) get(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Set("User-Agent", "Marathon-Consul")

	log.WithFields(log.Fields{
		"Uri":      request.URL.RequestURI(),
		"Location": m.Location,
		"Protocol": m.Protocol,
	}).Debug("Sending GET request to marathon")

	var response *http.Response
	metrics.Time("marathon.get", func() { response, err = m.client.Do(request) })
	if err != nil {
		metrics.Mark("marathon.get.error")
		m.logHTTPError(response, err)
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		metrics.Mark("marathon.get.error")
		metrics.Mark(fmt.Sprintf("marathon.get.error.%d", response.StatusCode))
		err = fmt.Errorf("Expected 200 but got %d for %s", response.StatusCode, response.Request.URL.Path)
		m.logHTTPError(response, err)
		return nil, err
	}

	return ioutil.ReadAll(response.Body)
}

func (m Marathon) logHTTPError(resp *http.Response, err error) {
	statusCode := "???"
	if resp != nil {
		statusCode = fmt.Sprintf("%d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"Location":   m.Location,
		"Protocol":   m.Protocol,
		"statusCode": statusCode,
	}).Error(err)
}

func (m Marathon) url(path string) string {
	return m.urlWithQuery(path, nil)
}

type params map[string][]string

func (m Marathon) urlWithQuery(path string, params params) string {
	marathon := url.URL{
		Scheme: m.Protocol,
		User:   m.Auth,
		Host:   m.Location,
		Path:   path,
	}
	query := marathon.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	marathon.RawQuery = query.Encode()
	return marathon.String()
}

func (m Marathon) AmILeader() (bool, error) {
	leader, err := m.Leader()
	return m.MyLeader == leader, err
}
