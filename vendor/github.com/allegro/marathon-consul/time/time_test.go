package time

import (
	"encoding/json"
	"testing"
	gotime "time"

	"github.com/stretchr/testify/assert"
)

func TestTimestampParsing(t *testing.T) {
	t.Parallel()
	in := "2014-03-01T23:29:30.158Z"
	bytes, err := json.Marshal(in)
	assert.NoError(t, err)
	var out Timestamp
	err = json.Unmarshal(bytes, &out)
	assert.NoError(t, err)
	assert.Equal(t, in, out.String())
}

func TestDurationIntegerParsing(t *testing.T) {
	t.Parallel()
	var out Interval
	err := json.Unmarshal([]byte("900000000000"), &out)
	assert.NoError(t, err)
	assert.Equal(t, "15m0s", out.String())
}

func TestInvalidDurationParsing(t *testing.T) {
	t.Parallel()
	var out Interval
	err := json.Unmarshal([]byte(`"72xyz"`), &out)
	assert.Error(t, err)
	assert.Equal(t, "0s", out.String())
}

func TestDurationMarshal(t *testing.T) {
	t.Parallel()
	var out Interval
	out.Duration, _ = gotime.ParseDuration("15m")
	bytes, err := json.Marshal(out)
	assert.NoError(t, err)
	err = json.Unmarshal(bytes, &out)
	assert.NoError(t, err)
	assert.Equal(t, "15m0s", out.String())
}
