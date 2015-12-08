package marathon

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestMarathon_AppsWhenMarathonReturnEmptyList(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps?embed=apps.tasks", `{"apps": []}`)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	apps, err := m.Apps()
	//then
	assert.NoError(t, err)
	assert.Empty(t, apps)
}

func TestMarathon_AppsWhenConfigIsWrong(t *testing.T) {
	t.Parallel()
	// given
	m, _ := New(Config{Location: "not::valid/location", Protocol: "HTTP"})
	// when
	apps, err := m.Apps()
	//then
	assert.Error(t, err)
	assert.Nil(t, apps)
}

func TestMarathon_AppsWhenServerIsNotResponding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	t.Parallel()
	// given
	m, _ := New(Config{Location: "unknown:22", Protocol: "HTTP"})
	// when
	apps, err := m.Apps()
	//then
	assert.Error(t, err)
	assert.Nil(t, apps)
}

func TestMarathon_AppsWhenMarathonConnectionFailedShouldRetry(t *testing.T) {
	t.Parallel()
	// given
	calls := 0
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(500)
	})
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	apps, err := m.Apps()
	//then
	assert.Error(t, err)
	assert.Empty(t, apps)
	assert.Equal(t, 3, calls)
}

func TestMarathon_TasksWhenMarathonConnectionFailedShouldRetry(t *testing.T) {
	t.Parallel()
	// given
	calls := 0
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(500)
	})
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	tasks, err := m.Tasks("/app/id")
	//then
	assert.Error(t, err)
	assert.Empty(t, tasks)
	assert.Equal(t, 3, calls)
}

func TestMarathon_AppWhenMarathonConnectionFailedShouldRetry(t *testing.T) {
	t.Parallel()
	// given
	calls := 0
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(500)
	})
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	app, err := m.App("/app/id")
	//then
	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Equal(t, 3, calls)
}

func TestMarathon_AppsWhenMarathonReturnEmptyResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps?embed=apps.tasks", ``)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	apps, err := m.Apps()
	//then
	assert.Nil(t, apps)
	assert.Error(t, err)
}

func TestMarathon_AppsWhenMarathonReturnMalformedJsonResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps?embed=apps.tasks", `{"apps":}`)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	app, err := m.App("/test/app")
	//then
	assert.Nil(t, app)
	assert.Error(t, err)
}

func TestMarathon_AppWhenMarathonReturnEmptyApp(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps//test/app?embed=apps.tasks", `{"app": {}}`)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	app, err := m.App("/test/app")
	//then
	assert.NoError(t, err)
	assert.NotNil(t, app)
}

func TestMarathon_AppWhenMarathonReturnEmptyResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps//test/app?embed=apps.tasks", ``)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	app, err := m.App("/test/app")
	//then
	assert.NotNil(t, app)
	assert.Error(t, err)
}

func TestMarathon_AppWhenMarathonReturnMalformedJsonResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps//test/app?embed=apps.tasks", `{apps:}`)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	apps, err := m.Apps()
	//then
	assert.Nil(t, apps)
	assert.Error(t, err)
}

func TestMarathon_TasksWhenMarathonReturnEmptyList(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps/test/app/tasks", `
	{"tasks": [{
		"appId": "/test",
		"host": "192.168.2.114",
		"id": "test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8",
		"ports": [31315],
		"healthCheckResults":[{ "alive":true }]
	}]}`)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	tasks, err := m.Tasks("//test/app")
	//then
	assert.NoError(t, err)
	assert.NotNil(t, tasks)
}

func TestMarathon_TasksWhenMarathonReturnEmptyResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps/test/app/tasks", ``)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	tasks, err := m.Tasks("/test/app")
	//then
	assert.Nil(t, tasks)
	assert.Error(t, err)
}

func TestMarathon_TasksWhenMarathonReturnMalformedJsonResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps/test/app/tasks", ``)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"})
	m.transport = transport
	// when
	tasks, err := m.Tasks("/test/app")
	//then
	assert.Nil(t, tasks)
	assert.Error(t, err)
}

func TestConfig_transport(t *testing.T) {
	t.Parallel()
	// given
	config := Config{VerifySsl: false}
	// when
	marathon, _ := New(config)
	// then
	transport, ok := marathon.transport.(*http.Transport)
	assert.True(t, ok)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestUrl_WithoutAuth(t *testing.T) {
	t.Parallel()
	// given
	config := Config{Location: "localhost:8080", Protocol: "http"}
	// when
	m, _ := New(config)
	// then
	assert.Equal(t, "http://localhost:8080/v2/apps", m.url("/v2/apps"))
}

func TestUrl_WithAuth(t *testing.T) {
	t.Parallel()
	// given
	config := Config{Location: "localhost:8080", Protocol: "http", Username: "peter", Password: "parker"}
	// when
	m, _ := New(config)
	// then
	assert.Equal(t, "http://peter:parker@localhost:8080/v2/apps", m.url("/v2/apps"))
}

// http://keighl.com/post/mocking-http-responses-in-golang/
func stubServer(uri string, body string) (*httptest.Server, *http.Transport) {
	return mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() == uri {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, body)
		} else {
			w.WriteHeader(404)
		}
	})
}

func mockServer(handle func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *http.Transport) {
	server := httptest.NewServer(http.HandlerFunc(handle))

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	return server, transport
}
