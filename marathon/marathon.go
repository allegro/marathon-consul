package marathon

import (
	"crypto/tls"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/metrics"
	"github.com/allegro/marathon-consul/tasks"
	"github.com/sethgrid/pester"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Marathoner interface {
	Apps() ([]*apps.App, error)
	App(string) (*apps.App, error)
	Tasks(string) ([]*tasks.Task, error)
}

type Marathon struct {
	Location  string
	Protocol  string
	Auth      *url.Userinfo
	transport http.RoundTripper
}

func New(config Config) (*Marathon, error) {
	var auth *url.Userinfo
	if len(config.Username) == 0 && len(config.Password) == 0 {
		auth = nil
	} else {
		auth = url.UserPassword(config.Username, config.Password)
	}
	return &Marathon{
		Location: config.Location,
		Protocol: config.Protocol,
		Auth:     auth,
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !config.VerifySsl,
			},
		},
	}, nil
}

func (m Marathon) App(appId string) (*apps.App, error) {
	log.WithField("Location", m.Location).Debug("Asking Marathon for " + appId)

	body, err := m.get(m.urlWithQuery("/v2/apps/"+appId, "embed=apps.tasks"))
	if err != nil {
		return nil, err
	}

	return apps.ParseApp(body)
}

func (m Marathon) Apps() ([]*apps.App, error) {
	log.WithField("Location", m.Location).Debug("Asking Marathon for apps")
	body, err := m.get(m.urlWithQuery("/v2/apps", "embed=apps.tasks"))
	if err != nil {
		return nil, err
	}

	return apps.ParseApps(body)
}

func (m Marathon) Tasks(app string) ([]*tasks.Task, error) {
	log.WithFields(log.Fields{
		"Location": m.Location,
		"Id":       app,
	}).Debug("asking Marathon for tasks")

	app = strings.Trim(app, "/")
	body, err := m.get(m.url(fmt.Sprintf("/v2/apps/%s/tasks", app)))
	if err != nil {
		return nil, err
	}

	return tasks.ParseTasks(body)
}

func (m Marathon) get(url string) ([]byte, error) {
	client := m.getClient()
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
	metrics.Time("marathon.get", func() { response, err = client.Do(request) })
	if err != nil {
		metrics.Mark("marathon.get.error")
		m.logHTTPError(response, err)
		return nil, err
	}
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
	return m.urlWithQuery(path, "")
}

func (m Marathon) urlWithQuery(path string, query string) string {
	marathon := url.URL{
		Scheme:   m.Protocol,
		User:     m.Auth,
		Host:     m.Location,
		Path:     path,
		RawQuery: query,
	}
	return marathon.String()
}

func (m Marathon) getClient() *pester.Client {
	client := pester.New()
	client.Transport = m.transport
	return client
}
