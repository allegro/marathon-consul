package consul

import (
	"fmt"
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"github.com/hashicorp/consul/api"
	"strings"
)

func WithPrefix(prefix, key string) string {
	if prefix != "" && !strings.HasPrefix(key, prefix) {
		key = fmt.Sprintf("%s/%s", prefix, key)
	}
	return key
}

func WithoutPrefix(prefix, key string) string {
	if prefix != "" && strings.HasPrefix(key, prefix) {
		key = key[len(prefix)+1 : len(key)]
	}
	return key
}

func MapKVPairs(source api.KVPairs) map[string]*api.KVPair {
	pairs := make(map[string]*api.KVPair, len(source))
	for _, pair := range source {
		pairs[pair.Key] = pair
	}
	return pairs
}

func MapApps(source []*apps.App) map[string]*api.KVPair {
	pairs := make(map[string]*api.KVPair, len(source))
	for _, app := range source {
		pair := app.KV()
		pairs[pair.Key] = pair
	}
	return pairs
}

func MapTasks(source []*tasks.Task) map[string]*api.KVPair {
	pairs := make(map[string]*api.KVPair, len(source))
	for _, app := range source {
		pair := app.KV()
		pairs[pair.Key] = pair
	}
	return pairs
}
