package marathon

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/metrics"
)

type Marathoner interface {
	ConsulApps() ([]*apps.App, error)
	App(apps.AppID) (*apps.App, error)
	Tasks(apps.AppID) ([]*apps.Task, error)
	Leader() (string, error)
}

type Marathon struct {
	Location string
	Protocol string
	Auth     *url.Userinfo
	client   *http.Client
}

type LeaderResponse struct {
	Leader string `json:"leader"`
}

func New(config Config) (*Marathon, error) {
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
	return &Marathon{
		Location: config.Location,
		Protocol: config.Protocol,
		Auth:     auth,
		client: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
		},
	}, nil
}

func (m Marathon) App(appId apps.AppID) (*apps.App, error) {
	log.WithField("Location", m.Location).Debug("Asking Marathon for " + appId)

	body, err := m.get(m.urlWithQuery(fmt.Sprintf("/v2/apps/%s", appId), params{"embed": "apps.tasks"}))
	if err != nil {
		return nil, err
	}

	return apps.ParseApp(body)
}

func (m Marathon) ConsulApps() ([]*apps.App, error) {
	log.WithField("Location", m.Location).Debug("Asking Marathon for apps")
	body, err := m.get(m.urlWithQuery("/v2/apps", params{"embed": "apps.tasks", "label": apps.MarathonConsulLabel}))
	if err != nil {
		return nil, err
	}

	return apps.ParseApps(body)
}

func (m Marathon) Tasks(app apps.AppID) ([]*apps.Task, error) {
	log.WithFields(log.Fields{
		"Location": m.Location,
		"Id":       app,
	}).Debug("asking Marathon for tasks")

	trimmedAppId := strings.Trim(app.String(), "/")
	body, err := m.get(m.url(fmt.Sprintf("/v2/apps/%s/tasks", trimmedAppId)))
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

func (m Marathon) get(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	request.Header.Add("Accept", "application/json")

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
	var statusCode string = "???"
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

type params map[string]string

func (m Marathon) urlWithQuery(path string, params params) string {
	marathon := url.URL{
		Scheme: m.Protocol,
		User:   m.Auth,
		Host:   m.Location,
		Path:   path,
	}
	query := marathon.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	marathon.RawQuery = query.Encode()
	return marathon.String()
}
