package consul

import (
	consulapi "github.com/hashicorp/consul/api"
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
	agent2, _ := agents.GetAgent("http://127.0.0.2")
	agent3, _ := agents.GetAgent("http://127.0.0.3")

	// when
	anyAgent, _ := agents.GetAnyAgent()

	// then
	assert.Contains(t, []*consulapi.Client{agent1, agent2, agent3}, anyAgent)
}

func TestGetAnyAgent_shouldFailOnNoAgentAvailable(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})

	// when
	anyAgent, err := agents.GetAnyAgent()

	// then
	assert.Nil(t, anyAgent)
	assert.NotNil(t, err)
}
