package mocks

import (
	"github.com/hashicorp/consul/api"
	"strings"
	"sync"
)

type KVer struct {
	KVs  map[string]*api.KVPair
	lock *sync.RWMutex
}

func NewKVer() KVer {
	return KVer{
		make(map[string]*api.KVPair),
		&sync.RWMutex{},
	}
}

func (kv KVer) Get(key string) (*api.KVPair, *api.QueryMeta, error) {
	kv.lock.RLock()
	defer kv.lock.RUnlock()

	return kv.KVs[key], &api.QueryMeta{}, nil
}

func (kv KVer) List(prefix string) (api.KVPairs, *api.QueryMeta, error) {
	kv.lock.RLock()
	defer kv.lock.RUnlock()

	kvs := api.KVPairs{}
	for key, value := range kv.KVs {
		if strings.HasPrefix(key, prefix) {
			kvs = append(kvs, value)
		}
	}
	return kvs, &api.QueryMeta{}, nil
}

func (kv KVer) Put(info *api.KVPair) (*api.WriteMeta, error) {
	kv.lock.Lock()
	defer kv.lock.Unlock()

	kv.KVs[info.Key] = info
	return &api.WriteMeta{}, nil
}

func (kv KVer) Delete(key string) (*api.WriteMeta, error) {
	kv.lock.Lock()
	defer kv.lock.Unlock()

	delete(kv.KVs, key)
	return &api.WriteMeta{}, nil
}
