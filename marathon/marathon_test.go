package marathon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarathon_AppsWhenMarathonReturnEmptyList(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps?embed=apps.tasks&label=consul", `{"apps": []}`)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
	// when
	apps, err := m.ConsulApps()
	//then
	assert.NoError(t, err)
	assert.Empty(t, apps)
}

func TestMarathon_AppsWhenConfigIsWrong(t *testing.T) {
	t.Parallel()
	// given
	m, _ := New(Config{Location: "not::valid/location", Protocol: "HTTP"}, "")
	// when
	apps, err := m.ConsulApps()
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
	m, _ := New(Config{Location: "unknown:22", Protocol: "HTTP"}, "")
	// when
	apps, err := m.ConsulApps()
	//then
	assert.Error(t, err)
	assert.Nil(t, apps)
}

func TestMarathon_AppsWhenMarathonConnectionFailedShouldNotRetry(t *testing.T) {
	t.Parallel()
	// given
	calls := 0
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(500)
	})
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
	// when
	apps, err := m.ConsulApps()
	//then
	assert.Error(t, err)
	assert.Empty(t, apps)
	assert.Equal(t, 1, calls)
}

func TestMarathon_TasksWhenMarathonConnectionFailedShouldNotRetry(t *testing.T) {
	t.Parallel()
	// given
	calls := 0
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(500)
	})
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
	// when
	tasks, err := m.Tasks("/app/id")
	//then
	assert.Error(t, err)
	assert.Empty(t, tasks)
	assert.Equal(t, 1, calls)
}

func TestMarathon_AppWhenMarathonConnectionFailedShouldNotRetry(t *testing.T) {
	t.Parallel()
	// given
	calls := 0
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(500)
	})
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
	// when
	app, err := m.App("/app/id")
	//then
	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Equal(t, 1, calls)
}

func TestMarathon_AppsWhenMarathonReturnEmptyResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps?label=consul&embed=apps.tasks", ``)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
	// when
	apps, err := m.ConsulApps()
	//then
	assert.Nil(t, apps)
	assert.Error(t, err)
}

func TestMarathon_AppsWhenMarathonReturnMalformedJsonResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps?label=consul&embed=apps.tasks", `{"apps":}`)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
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
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
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
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
	// when
	app, err := m.App("/test/app")
	//then
	assert.NotNil(t, app)
	assert.Error(t, err)
}

func TestMarathon_AppWhenMarathonReturnMalformedJsonResponse(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/apps//test/app?label=consul&embed=apps.tasks", `{apps:}`)
	defer server.Close()

	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
	// when
	apps, err := m.ConsulApps()
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
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
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
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
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
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport
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
	marathon, _ := New(config, "")
	// then
	transport, ok := marathon.client.Transport.(*http.Transport)
	assert.True(t, ok)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestUrl_WithoutAuth(t *testing.T) {
	t.Parallel()
	// given
	config := Config{Location: "localhost:8080", Protocol: "http"}
	// when
	m, _ := New(config, "")
	// then
	assert.Equal(t, "http://localhost:8080/v2/apps", m.url("/v2/apps"))
}

func TestUrl_WithAuth(t *testing.T) {
	t.Parallel()
	// given
	config := Config{Location: "localhost:8080", Protocol: "http", Username: "peter", Password: "parker"}
	// when
	m, _ := New(config, "")
	// then
	assert.Equal(t, "http://peter:parker@localhost:8080/v2/apps", m.url("/v2/apps"))
}

func TestLeader_SuccessfulResponse(t *testing.T) {
	t.Parallel()

	// given
	server, transport := stubServer("/v2/leader", `{"leader": "some.leader.host:8081"}`)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport

	// when
	leader, err := m.Leader()

	//then
	assert.NoError(t, err)
	assert.Equal(t, "some.leader.host:8081", leader)
}

func TestLeader_ErrorOnMalformedJsonResponse(t *testing.T) {
	t.Parallel()

	// given
	server, transport := stubServer("/v2/leader", "{")
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport

	// when
	leader, err := m.Leader()

	//then
	assert.Error(t, err)
	assert.Empty(t, leader)
}

func TestLeader_NotRetryOnFailingResponse(t *testing.T) {
	t.Parallel()

	// given
	calls := 0
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(500)
	})
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "")
	m.client.Transport = transport

	// when
	leader, err := m.Leader()

	//then
	assert.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.Empty(t, leader)
}

func TestLeaderPoll_PassingRunningOnLeader(t *testing.T) {
	t.Parallel()

	// given
	server, transport := stubServer("/v2/leader", `{"leader": "this.leader:8080"}`)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "this.leader:8080")
	m.client.Transport = transport

	// when
	err := m.leaderPoll()

	//then
	assert.NoError(t, err)
}

func TestAMILeader_PassingRunningOnLeader(t *testing.T) {
	t.Parallel()

	// given
	server, transport := stubServer("/v2/leader", `{"leader": "this.leader:8080"}`)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "this.leader:8080")
	m.client.Transport = transport

	// when
	leading, err := m.AmILeader()

	//then
	assert.True(t, leading)
	assert.NoError(t, err)
}

func TestAMILeader_NotPassingNotRunningOnLeader(t *testing.T) {
	t.Parallel()

	// given
	server, transport := stubServer("/v2/leader", `{"leader": "other.leader:8080"}`)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "this.leader:8080")
	m.client.Transport = transport

	// when
	leading, err := m.AmILeader()

	//then
	assert.False(t, leading)
	assert.NoError(t, err)
}

func TestEventStream_PassingStreamerCreated(t *testing.T) {
	t.Parallel()

	// given
	server, transport := stubServer("/v2/leader", `{"leader": "this.leader:8080"}`)
	defer server.Close()
	url, _ := url.Parse(server.URL)
	m, _ := New(Config{Location: url.Host, Protocol: "HTTP"}, "this.leader:8080")
	m.client.Transport = transport

	// when
	streamer, err := m.EventStream([]string{}, 1, 1)

	//then
	assert.NoError(t, err)
	assert.IsType(t, &Streamer{}, streamer)
}

func TestUrlWithQuery_NoProxyMarathon(t *testing.T) {
	t.Parallel()

	// given
	m, _ := New(Config{Location: "localhost:8080", Protocol: "HTTP"}, "")
	// when
	path := m.urlWithQuery("/testpath", params{})

	// then
	assert.Equal(t, "HTTP://localhost:8080/testpath", path)
}

func TestUrlWithQuery_ProxyMarathon(t *testing.T) {
	t.Parallel()

	// given
	m, _ := New(Config{Location: "localhost:8080/proxy/url/segments", Protocol: "HTTP"}, "")
	// when
	path := m.urlWithQuery("/testpath", params{})

	// then
	assert.Equal(t, "HTTP://localhost:8080/proxy/url/segments/testpath", path)
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
