package consul

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})

	// when
	agent, _ := agents.GetAgent("http://127.0.0.1")

	// then
	assert.NotNil(t, agent)
}

func TestGetAnyAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})
	agent1, _ := agents.GetAgent("http://127.0.0.1")

	// when
	agent2, _ := agents.GetAnyAgent()

	// then
	assert.Equal(t, agent1, agent2)
}
