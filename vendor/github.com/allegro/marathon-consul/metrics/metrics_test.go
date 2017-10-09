package metrics

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestMark(t *testing.T) {
	// given
	Init(Config{Target: "stdout", Prefix: ""})

	// expect
	assert.Nil(t, metrics.Get("marker"))

	// when
	Mark("marker")

	// then
	mark, _ := metrics.Get("marker").(metrics.Meter)
	assert.Equal(t, int64(1), mark.Count())

	// when
	Mark("marker")

	// then
	assert.Equal(t, int64(2), mark.Count())

	// when
	Clear()

	// then
	assert.Nil(t, metrics.Get("marker"))
}

func TestTime(t *testing.T) {
	// given
	Init(Config{Target: "stdout", Prefix: ""})

	// expect
	assert.Nil(t, metrics.Get("timer"))

	// when
	Time("timer", func() {})

	// then
	time, _ := metrics.Get("timer").(metrics.Timer)
	assert.Equal(t, int64(1), time.Count())

	// when
	Time("timer", func() {})

	// then
	assert.Equal(t, int64(2), time.Count())

	// when
	Clear()

	// then
	assert.Nil(t, metrics.Get("marker"))
}

func TestUpdateGauge(t *testing.T) {
	// given
	Init(Config{Target: "stdout", Prefix: ""})

	// expect
	assert.Nil(t, metrics.Get("counter"))

	// when
	UpdateGauge("counter", 2)

	// then
	gauge := metrics.Get("counter").(metrics.Gauge)
	assert.Equal(t, int64(2), gauge.Value())

	// when
	UpdateGauge("counter", 123)

	// then
	assert.Equal(t, int64(123), gauge.Value())

	// when
	Clear()

	// then
	assert.Nil(t, metrics.Get("marker"))
}

func TestMetricsInit_ForGraphiteWithNoAddress(t *testing.T) {
	err := Init(Config{Target: "graphite", Addr: ""})
	assert.Error(t, err)
}

func TestMetricsInit_ForGraphiteWithBadAddress(t *testing.T) {
	err := Init(Config{Target: "graphite", Addr: "localhost"})
	assert.Error(t, err)
}

func TestMetricsInit_ForGraphit(t *testing.T) {
	err := Init(Config{Target: "graphite", Addr: "localhost:81"})
	assert.NoError(t, err)
}

func TestMetricsInit_ForUnknownTarget(t *testing.T) {
	err := Init(Config{Target: "unknown"})
	assert.Error(t, err)
}

func TestMetricsInit(t *testing.T) {
	// when
	err := Init(Config{Prefix: "prefix"})

	// then
	assert.Equal(t, "prefix", pfx)
	assert.NoError(t, err)
}

func TestInit_DefaultPrefix(t *testing.T) {
	// given
	hostname = func() (string, error) { return "", fmt.Errorf("Some error") }

	// when
	err := Init(Config{Prefix: "default"})

	// then
	assert.Error(t, err)
}

func TestInit_DefaultPrefix_WithErrors(t *testing.T) {
	// given
	hostname = func() (string, error) { return "myhost", nil }
	os.Args = []string{"./myapp"}

	// when
	err := Init(Config{Prefix: "default"})

	// then
	assert.NoError(t, err)
	assert.Equal(t, "myhost.myapp", pfx)
}

func TestTargetName(t *testing.T) {
	tests := []struct {
		service, host, path, target string
		name                        string
	}{
		{"s", "h", "p", "http://foo.com/bar", "s.h.p.foo_com"},
		{"s", "", "p", "http://foo.com/bar", "s._.p.foo_com"},
		{"s", "", "", "http://foo.com/bar", "s._._.foo_com"},
		{"", "", "", "http://foo.com/bar", "_._._.foo_com"},
		{"", "", "", "http://foo.com:1234/bar", "_._._.foo_com_1234"},
		{"", "", "", "http://1.2.3.4:1234/bar", "_._._.1_2_3_4_1234"},
	}

	for i, tt := range tests {
		u, err := url.Parse(tt.target)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if got, want := TargetName(tt.service, tt.host, tt.path, u), tt.name; got != want {
			t.Errorf("%d: got %q want %q", i, got, want)
		}
	}
}
