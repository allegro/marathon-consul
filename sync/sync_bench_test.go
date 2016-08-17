package sync

import (
	"fmt"
	"testing"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/service"
	. "github.com/allegro/marathon-consul/utils"
)

func BenchmarkDeregisterConsulServicesThatAreNotInMarathonApps10x2(b *testing.B) {
	const (
		appsCount      = 10
		instancesCount = 2
	)

	bench(b, appsCount, instancesCount)
}

func BenchmarkDeregisterConsulServicesThatAreNotInMarathonApps100x2(b *testing.B) {
	const (
		appsCount      = 100
		instancesCount = 2
	)

	bench(b, appsCount, instancesCount)
}

func BenchmarkDeregisterConsulServicesThatAreNotInMarathonApps100x100(b *testing.B) {
	const (
		appsCount      = 100
		instancesCount = 100
	)

	bench(b, appsCount, instancesCount)
}

func bench(b *testing.B, appsCount, instancesCount int) {
	apps := marathonApps(appsCount, instancesCount)
	instances := instances(appsCount, instancesCount)
	sync := New(Config{}, nil, consul.NewConsulStub(), noopSyncStartedListener)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sync.deregisterConsulServicesNotFoundInMarathon(apps, instances)
	}
}

func marathonApps(appsCount, instancesCount int) []*apps.App {
	marathonApps := make([]*apps.App, appsCount)
	for i := 0; i < appsCount; i++ {
		marathonApps[i] = ConsulApp(fmt.Sprintf("marathon/app/no_%d", i), instancesCount)
	}
	return marathonApps
}

func instances(appsCount, instancesCount int) []*service.Service {
	createdInstances := make([]*service.Service, appsCount*instancesCount)
	for i := 0; i < appsCount*instancesCount; i++ {
		app := ConsulApp(fmt.Sprintf("consul/service/no_%d", i), instancesCount)
		for _, task := range app.Tasks {
			createdInstances[i] = &service.Service{
				ID:   service.ServiceId(task.ID.String()),
				Name: app.ID.String(),
				Tags: []string{"marathon"},
				RegisteringAgentAddress: task.Host,
			}
		}
	}
	return createdInstances
}
