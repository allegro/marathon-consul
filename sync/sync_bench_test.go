package sync

import (
	"fmt"
	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/consul"
	. "github.com/allegro/marathon-consul/utils"
	consulapi "github.com/hashicorp/consul/api"
	"testing"
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
	instances := consulInstances(appsCount, instancesCount)
	sync := New(nil, consul.NewConsulStub())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sync.deregisterConsulServicesThatAreNotInMarathonApps(apps, instances)
	}
}

func marathonApps(appsCount, instancesCount int) []*apps.App {
	marathonApps := make([]*apps.App, appsCount)
	for i := 0; i < appsCount; i++ {
		marathonApps[i] = ConsulApp(fmt.Sprintf("marathon/app/no_%d", i), instancesCount)
	}
	return marathonApps
}

func consulInstances(appsCount, instancesCount int) []*consulapi.CatalogService {
	consulServices := make([]*consulapi.CatalogService, appsCount*instancesCount)
	for i := 0; i < appsCount*instancesCount; i++ {
		app := ConsulApp(fmt.Sprintf("consul/service/no_%d", i), instancesCount)
		for _, task := range app.Tasks {
			consulServices[i] = &consulapi.CatalogService{
				Address:        task.Host,
				ServiceAddress: task.Host,
				ServicePort:    task.Ports[0],
				ServiceTags:    []string{"marathon"},
				ServiceID:      task.ID,
				ServiceName:    app.ID,
			}
		}
	}
	return consulServices
}
