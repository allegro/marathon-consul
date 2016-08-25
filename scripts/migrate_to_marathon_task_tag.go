package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

type Config struct {
	BootstrapAgentAddress string
	ConsulTag             string
}

type MigrationStats struct {
	MigratedServices                          int
	SkippedFailedNodes                        int
	SkippedFailedServices                     int
	SkippedServicesNotManagedByMarathonConsul int
	SkippedServicesAlreadyWithMarathonTaskTag int
}

func main() {
	config := createConfig()
	migrateSingleDC(config.BootstrapAgentAddress, config.ConsulTag)
}

func createConfig() *Config {
	config := &Config{}
	flag.StringVar(&config.BootstrapAgentAddress, "bootstrap-agent-address", "127.0.0.1:8500", "Address to one of the agents in the migrated DC")
	flag.StringVar(&config.ConsulTag, "consul-tag", "marathon", "Tag specifying that a service is managed by marathon-consul")
	flag.Parse()
	return config
}

func migrateSingleDC(bootstrapAgentAddress string, consulTag string) *MigrationStats {
	logrus.WithField("BootstrapAgentAddress", bootstrapAgentAddress).WithField("Tag", consulTag).Info("Starting migration...")

	agentsPort, err := extractPort(bootstrapAgentAddress)
	if err != nil {
		logrus.WithError(err).WithField("BootstrapAgentAddress", bootstrapAgentAddress).
			Error("Could not extract port from agent address, aborting.")
		os.Exit(1)
	}

	client, err := api.NewClient(&api.Config{
		Address: bootstrapAgentAddress,
	})
	if err != nil {
		logrus.WithError(err).WithField("BootstrapAgentAddress", bootstrapAgentAddress).
			Error("Could not create client to agent, aborting.")
		os.Exit(1)
	}

	nodes, _, err := client.Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		logrus.WithError(err).Error("Could not fetch nodes, aborting.")
		os.Exit(1)
	}

	logrus.Infof("Discovered %d node(s) to migrate", len(nodes))
	stats := migrateNodes(nodes, agentsPort, consulTag)
	logrus.WithField("Stats", stats).Info("Migration finished")
	return stats
}

func migrateNodes(nodes []*api.Node, agentsPort int, consulTag string) *MigrationStats {
	stats := &MigrationStats{}

	for _, node := range nodes {
		logrus.WithField("Node", node.Node).Info("Migrating node...")
		nodeAddress := fmt.Sprintf("%s:%d", node.Address, agentsPort)
		nodeClient, err := api.NewClient(&api.Config{
			Address: nodeAddress,
		})
		if err != nil {
			logrus.WithError(err).WithField("Node", node.Node).WithField("Address", nodeAddress).
				Warn("Could not create client to node, skipping")
			stats.SkippedFailedNodes++
			continue
		}
		agentServices, err := nodeClient.Agent().Services()
		if err != nil {
			logrus.WithError(err).WithField("Node", node.Node).Warn("Could not fetch services on node, skipping")
			stats.SkippedFailedNodes++
			continue
		}

		migrateServicesOnNode(agentServices, nodeClient, consulTag, stats)
		logrus.WithField("Node", node.Node).Info("Migrated node")
	}

	return stats
}

func migrateServicesOnNode(services map[string]*api.AgentService, nodeClient *api.Client, consulTag string, stats *MigrationStats) {
	for _, agentService := range services {
		if !isManagedByMarathonConsul(agentService, consulTag) {
			logrus.WithField("ServiceID", agentService.ID).Info("Service not managed by marathon-consul, skipping")
			stats.SkippedServicesNotManagedByMarathonConsul++
			continue
		}
		if hasMarathonTaskTag(agentService) {
			logrus.WithField("ServiceID", agentService.ID).Info("Service already has marathon-task tag, skipping")
			stats.SkippedServicesAlreadyWithMarathonTaskTag++
			continue
		}

		err := nodeClient.Agent().ServiceRegister(migrated(agentService))
		if err != nil {
			logrus.WithError(err).WithField("ServiceID", agentService.ID).
				Warn("Could not reregister service, skipping")
			stats.SkippedFailedServices++
			continue
		}
		logrus.WithField("ServiceID", agentService.ID).Info("Migrated service")
		stats.MigratedServices++
	}
}

func extractPort(address string) (int, error) {
	return strconv.Atoi(address[strings.LastIndex(address, ":")+1:])
}

func isManagedByMarathonConsul(checked *api.AgentService, consulTag string) bool {
	for _, tag := range checked.Tags {
		if tag == consulTag {
			return true
		}
	}
	return false
}

func hasMarathonTaskTag(checked *api.AgentService) bool {
	for _, tag := range checked.Tags {
		if strings.HasPrefix(tag, "marathon-task:") {
			return true
		}
	}
	return false
}

func migrated(toMigrate *api.AgentService) *api.AgentServiceRegistration {
	tags := append(toMigrate.Tags, fmt.Sprintf("marathon-task:%s", toMigrate.ID))

	return &api.AgentServiceRegistration{
		ID:                toMigrate.ID,
		Name:              toMigrate.Service,
		Tags:              tags,
		Port:              toMigrate.Port,
		Address:           toMigrate.Address,
		EnableTagOverride: toMigrate.EnableTagOverride,
	}
}
