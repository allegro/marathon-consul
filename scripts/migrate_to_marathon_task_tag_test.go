package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/allegro/marathon-consul/consul"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMigrateToMarathonTaskTag_shouldMigrateOnlyServicesWithMarathonTagAndWithoutTaskTag(t *testing.T) {
	t.Parallel()
	// given
	server := consul.CreateConsulTestServer(t)
	defer server.Stop()
	server.AddService("serviceA", "passing", []string{"marathon"})
	server.AddService("serviceB-critical", "critical", []string{"marathon"})
	server.AddService("serviceC-with-marathon-task-tag", "passing", []string{"marathon", "marathon-task:some-task"})
	server.AddService("serviceD-without-marathon-tag", "passing", []string{})

	// when
	stats := migrateSingleDC(server.HTTPAddr, "marathon")

	// then
	client, _ := clientToServer(server)
	services, _, _ := client.Catalog().Services(&api.QueryOptions{})
	assert.Contains(t, services["serviceA"], "marathon-task:serviceA")
	assert.Contains(t, services["serviceB-critical"], "marathon-task:serviceB-critical")
	assert.NotContains(t, services["serviceC-with-marathon-task-tag"], "marathon-task:serviceC-with-marathon-task-tag")
	assert.NotContains(t, services["serviceD-without-marathon-tag"], "marathon-task:serviceD-without-marathon-tag")
	assert.Equal(t, 2, stats.MigratedServices)
	assert.Equal(t, 2, stats.SkippedServicesNotManagedByMarathonConsul) // +1 for service consul
	assert.Equal(t, 1, stats.SkippedServicesAlreadyWithMarathonTaskTag)
}

func TestMigrateToMarathonTaskTag_shouldLeaveServicePropertiesOtherThanTagsUnchanged(t *testing.T) {
	t.Parallel()
	// given
	server := consul.CreateConsulTestServer(t)
	defer server.Stop()
	client, _ := clientToServer(server)
	client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      "serviceA.0",
		Name:    "serviceA",
		Tags:    []string{"marathon"},
		Port:    1337,
		Address: "1.2.3.4",
		Check: &api.AgentServiceCheck{
			HTTP:     "http://1.2.3.4/health",
			Interval: "60s",
		},
	})
	entriesBeforeMigration, _, _ := client.Health().Service("serviceA", "", false, &api.QueryOptions{})
	before := entriesBeforeMigration[0]

	// when
	migrateSingleDC(server.HTTPAddr, "marathon")

	// then
	entriesAfterMigration, _, _ := client.Health().Service("serviceA", "", false, &api.QueryOptions{})
	entryAfterMigrationAsJSON, _ := json.Marshal(entriesAfterMigration[0])
	expectedEntryAfterMigrationAsJSON, _ := json.Marshal(&api.ServiceEntry{
		Node:   before.Node,
		Checks: before.Checks,
		Service: &api.AgentService{
			ID:                before.Service.ID,
			Service:           before.Service.Service,
			Tags:              []string{"marathon", fmt.Sprintf("marathon-task:%s", before.Service.ID)},
			Port:              before.Service.Port,
			Address:           before.Service.Address,
			EnableTagOverride: before.Service.EnableTagOverride,
		},
	})
	assert.Equal(t, string(expectedEntryAfterMigrationAsJSON), string(entryAfterMigrationAsJSON))
}

func TestMigrateToMarathonTaskTag_shouldMigrateServicesInBootstrapAgentsDCOnly(t *testing.T) {
	t.Parallel()
	// given
	server := consul.CreateConsulTestServer(t)
	defer server.Stop()
	serverOnOtherDC := consul.CreateConsulTestServer(t)
	defer serverOnOtherDC.Stop()
	server.JoinWAN(serverOnOtherDC.WANAddr)

	server.AddService("serviceA", "passing", []string{"marathon"})
	serverOnOtherDC.AddService("serviceB", "passing", []string{"marathon"})

	// when
	migrateSingleDC(server.HTTPAddr, "marathon")

	// then
	client, _ := clientToServer(server)
	clientToOtherDC, _ := clientToServer(serverOnOtherDC)
	services, _, _ := client.Catalog().Services(&api.QueryOptions{})
	servicesOnOtherDC, _, _ := clientToOtherDC.Catalog().Services(&api.QueryOptions{})
	assert.Contains(t, services["serviceA"], "marathon-task:serviceA")
	assert.NotContains(t, servicesOnOtherDC["serviceB"], "marathon-task:serviceC")
}

func clientToServer(server *testutil.TestServer) (*api.Client, error) {
	return api.NewClient(&api.Config{
		Address:    server.HTTPAddr,
		Datacenter: server.Config.Datacenter,
	})
}
