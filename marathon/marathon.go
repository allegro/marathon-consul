package marathon

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/metrics"
)

var hostname = os.Hostname

type Marathoner interface {
	ConsulApps() ([]*apps.App, error)
	App(apps.AppID) (*apps.App, error)
	Tasks(apps.AppID) ([]apps.Task, error)
	Leader() (string, error)
	EventStream([]string, int, time.Duration) (*Streamer, error)
	IsLeader() (bool, error)
}

type Marathon struct {
	Location string
	Protocol string
	MyLeader string
	username string
	password string
	client   *http.Client
}

type LeaderResponse struct {
	Leader string `json:"leader"`
}

func New(config Config) (*Marathon, error) {
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
		MyLeader: config.Leader,
		username: config.Username,
		password: config.Password,
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
func (m Marathon) EventStream(desiredEvents []string, retries int, retryBackoff time.Duration) (*Streamer, error) {
	subURL := m.urlWithQuery("/v2/events", params{"event_type": desiredEvents})

	// Before creating actual streamer, this function blocks until configured leader for this receiver is elected.
	// When leaderPoll function successfully exit this instance of marathon-consul,
	// consider itself as a new leader and initializes Streamer.
	if err := m.leaderPoll(); err != nil {
		return nil, fmt.Errorf("Leader poll failed: %s", err)
	}

	return &Streamer{
		subURL:   subURL,
		username: m.username,
		password: m.password,
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
	pollTicker := time.NewTicker(1 * time.Second)
	defer pollTicker.Stop()
	retries := 5
	i := 0
	for range pollTicker.C {
		leading, err := m.IsLeader()
		if err != nil {
			if i >= retries {
				return fmt.Errorf("Failed to get a leader after %d retries", i)
			}
			i++
			continue
		}
		if leading {
			metrics.UpdateGauge("leader", int64(1))
			return nil
		}
		metrics.UpdateGauge("leader", int64(0))
	}
	return nil
}

func (m Marathon) get(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Set("User-Agent", "Marathon-Consul")

	log.WithFields(log.Fields{
		"Uri":      request.URL.RequestURI(),
		"Location": m.Location,
		"Protocol": m.Protocol,
	}).Debug("Sending GET request to marathon")

	request.SetBasicAuth(m.username, m.password)
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
	}).WithError(err).Warning("Error on http request")
}

func (m Marathon) url(path string) string {
	return m.urlWithQuery(path, nil)
}

type params map[string][]string

// urlWithQuery returns absolute path to marathon endpoint
// if location is given with path e.g. "localhost:8080/proxy/url", then
// host and path parts are appended to respective url.URL fields
func (m Marathon) urlWithQuery(path string, params params) string {
	var marathon url.URL
	if strings.Contains(m.Location, "/") {
		parts := strings.SplitN(m.Location, "/", 2)
		marathon = url.URL{
			Scheme: m.Protocol,
			Host:   parts[0],
			Path:   "/" + parts[1] + path,
		}
	} else {
		marathon = url.URL{
			Scheme: m.Protocol,
			Host:   m.Location,
			Path:   path,
		}
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

func (m *Marathon) IsLeader(port uint32) (bool, error) {
	if m.MyLeader == "*" {
		log.Debug("Leader detection disable")
		return true, nil
	}
	if m.MyLeader == "" {
		if err := m.resolveHostname(port); err != nil {
			return false, fmt.Errorf("Could not resolve hostname: %v", err)
		}
	}
	leader, err := m.Leader()
	return m.MyLeader == leader, err
}

func (m *Marathon) resolveHostname(port uint32) error {
	hostname, err := hostname()
	if err != nil {
		return err
	}
	m.MyLeader = fmt.Sprintf("%s:%s", hostname, port)
	log.WithField("Leader", m.MyLeader).Info("Marathon Leader mode set to resolved hostname")
	return nil
}
