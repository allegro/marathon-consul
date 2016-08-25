package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/Sirupsen/logrus"
)

type Config struct {
	BootstrapAgentAddress string
	ConsulTag             string
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
// TODO logs and stats

func migrateSingleDC(bootstrapAgentAddress string, consulTag string) {
	agentsPort, err := extractPort(bootstrapAgentAddress)
	if err != nil {
		logrus.WithError(err).WithField("AgentAddress", bootstrapAgentAddress).
			Error("Could not extract port from bootstrap agent address, aborting.")
		os.Exit(1)
	}

	client, err := api.NewClient(&api.Config{
		Address: bootstrapAgentAddress,
	})
	if err != nil {
		logrus.WithError(err).WithField("AgentAddress", bootstrapAgentAddress).
			Error("Could not create client to agent, aborting.")
		os.Exit(1)
	}

	nodes, _, err := client.Catalog().Nodes(&api.QueryOptions{})
	if err != nil {
		logrus.WithError(err).Error("Could not fetch nodes, aborting.")
		os.Exit(1)
	}

	migrateNodes(nodes, agentsPort, consulTag)
}

func migrateNodes(nodes []*api.Node, agentsPort int, consulTag string) {
	for _, node := range nodes {
		nodeAddress := fmt.Sprintf("%s:%d", node.Address, agentsPort)
		nodeClient, err := api.NewClient(&api.Config{
			Address: nodeAddress,
		})
		if err != nil {
			logrus.WithError(err).WithField("Node", node.Node).WithField("Address", nodeAddress).
				Warn("Could not create client to node, skipping this node")
			continue
		}
		agentServices, err := nodeClient.Agent().Services()
		if err != nil {
			logrus.WithError(err).WithField("Node", node.Node).Warn("Could not fetch services, skipping this node")
			continue
		}

		migrateServicesOnNode(agentServices, nodeClient, consulTag)
	}
}

func migrateServicesOnNode(services map[string]*api.AgentService, nodeClient *api.Client, consulTag string) {
	for _, agentService := range services {
		if shouldBeMigrated(agentService, consulTag) {
			err := nodeClient.Agent().ServiceRegister(migrated(agentService))
			if err != nil {
				logrus.WithError(err).WithField("ServiceID", agentService.ID).
					Warn("Could not reregister service, skipping this service")
				continue
			}
			logrus.WithField("ServiceID", agentService.ID).Info("Migrated service")
		}
	}
}

func extractPort(address string) (int, error) {
	return strconv.Atoi(address[strings.LastIndex(address, ":")+1:])
}

func shouldBeMigrated(checked *api.AgentService, consulTag string) bool {
	foundMarathonTag := false
	foundTaskTag := false

	for _, tag := range checked.Tags {
		if tag == consulTag {
			foundMarathonTag = true
		} else if strings.HasPrefix(tag, "marathon-task:") {
			foundTaskTag = true
		}
	}
	return foundMarathonTag && !foundTaskTag
}

func migrated(toMigrate *api.AgentService) *api.AgentServiceRegistration {
	tags := append(toMigrate.Tags, fmt.Sprintf("marathon-task:%s", toMigrate.ID))

	return &api.AgentServiceRegistration{
		ID: toMigrate.ID,
		Name: toMigrate.Service,
		Tags: tags,
		Port: toMigrate.Port,
		Address: toMigrate.Address,
		EnableTagOverride: toMigrate.EnableTagOverride,
	}
}
