package consul

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})

	// when
	agent, err := agents.GetAgent("127.0.0.1")

	// then
	assert.NotNil(t, agent)
	assert.NoError(t, err)
}

func TestPopulateAgentsCacheWithLocalAgent(t *testing.T) {
	t.Parallel()
	// when
	agents := NewAgents(&Config{LocalAgentHost: "127.0.0.1"})

	// then
	assert.Len(t, agents.agents, 1)
	assert.NotNil(t, agents.agents["127.0.0.1"])
}

func TestGetAnyAgent_SingleAgentAvailable(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})

	// when
	agents.GetAgent("127.0.0.1") // create
	agent, err := agents.GetAnyAgent()

	// then
	assert.NotNil(t, agent)
	assert.Equal(t, "127.0.0.1", agent.IP)
	assert.NoError(t, err)
}

func TestGetAnyAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})
	agent1, _ := agents.GetAgent("127.0.0.1")
	agent2, _ := agents.GetAgent("127.0.0.2")
	agent3, _ := agents.GetAgent("127.0.0.3")

	// when
	anyAgent, err := agents.GetAnyAgent()

	// then
	assert.NoError(t, err)
	assert.Contains(t, []*Agent{agent1, agent2, agent3}, anyAgent)
}

func TestGetAgent_ShouldResolveAddressToIP(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})

	// when
	agent1, _ := agents.GetAgent("127.0.0.1")
	agent2, _ := agents.GetAgent("localhost")

	// then
	assert.Equal(t, agent1, agent2)
}

func TestGetAgent_ShouldFailOnWrongHostname(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})

	// when
	_, err := agents.GetAgent("wrong hostname")

	// then
	assert.Error(t, err)
}

func TestGetAgent_ShouldFailOnUnknownHostname(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})

	// when
	_, err := agents.GetAgent("unknown.host.name.1")

	// then
	assert.Error(t, err)
}

func TestGetAnyAgent_shouldFailOnNoAgentAvailable(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})

	// when
	anyAgent, err := agents.GetAnyAgent()

	// then
	assert.Nil(t, anyAgent)
	assert.NotNil(t, err)
}

func TestRemoveAgent(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})
	agents.GetAgent("127.0.0.1")
	agent2, _ := agents.GetAgent("127.0.0.2")

	// when
	agents.RemoveAgent("127.0.0.1")

	// then
	for i := 0; i < 10; i++ {
		agent, err := agents.GetAnyAgent()
		assert.Equal(t, agent, agent2)
		assert.Equal(t, "127.0.0.2", agent.IP)
		assert.NoError(t, err)
	}
}

func TestRemoveAgentTwiceShouldPass(t *testing.T) {
	t.Parallel()
	// given
	agents := NewAgents(&Config{})
	agents.GetAgent("127.0.0.1")

	// when
	agents.RemoveAgent("127.0.0.1")
	agents.RemoveAgent("127.0.0.1")

	// then
	assert.Empty(t, agents.agents)
}
