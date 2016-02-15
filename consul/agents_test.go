package consul

import (
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestGetAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})

	// when
	agent, err := agents.GetAgent("127.0.0.1")

	// then
	assert.NotNil(t, agent)
	assert.NoError(t, err)
}

func TestGetAnyAgent_SingleAgentAvailable(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})

	// when
	agents.GetAgent("127.0.0.1") // create
	agent, address, err := agents.GetAnyAgent()

	// then
	assert.NotNil(t, agent)
	assert.Equal(t, "127.0.0.1", address)
	assert.NoError(t, err)
}

func TestGetAnyAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})
	agent1, _ := agents.GetAgent("127.0.0.1")
	agent2, _ := agents.GetAgent("127.0.0.2")
	agent3, _ := agents.GetAgent("127.0.0.3")

	// when
	anyAgent, _, _ := agents.GetAnyAgent()

	// then
	assert.Contains(t, []*consulapi.Client{agent1, agent2, agent3}, anyAgent)
}

func TestGetAgent_ShouldResolveAddressToIP(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})

	// when
	agent1, _ := agents.GetAgent("127.0.0.1")
	agent2, _ := agents.GetAgent("localhost")

	// then
	assert.Equal(t, agent1, agent2)
}

func TestGetAnyAgent_shouldFailOnNoAgentAvailable(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})

	// when
	anyAgent, _, err := agents.GetAnyAgent()

	// then
	assert.Nil(t, anyAgent)
	assert.NotNil(t, err)
}

func TestRemoveAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&ConsulConfig{})
	agents.GetAgent("127.0.0.1")
	agent2, _ := agents.GetAgent("127.0.0.2")

	// when
	agents.RemoveAgent("127.0.0.1")

	// then
	for i := 0; i < 10; i++ {
		agent, address, err := agents.GetAnyAgent()
		assert.Equal(t, agent, agent2)
		assert.Equal(t, "127.0.0.2", address)
		assert.NoError(t, err)
	}
}
