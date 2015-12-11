package web

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "http://example.com/health", bytes.NewBuffer([]byte{}))
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	HealthHandler(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}
