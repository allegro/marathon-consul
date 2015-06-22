package consul

import (
	"github.com/hashicorp/consul/api"
)

type Getter interface {
	Get(string) (*api.KVPair, *api.QueryMeta, error)
}

type Lister interface {
	List(string) (api.KVPairs, *api.QueryMeta, error)
}

type Putter interface {
	Put(*api.KVPair) (*api.WriteMeta, error)
}

type Deleter interface {
	Delete(string) (*api.WriteMeta, error)
}

type KVer interface {
	Getter
	Lister
	Putter
	Deleter
}

type KV struct {
	kv           *api.KV
	WriteOptions *api.WriteOptions
	QueryOptions *api.QueryOptions
	Prefix       string
}

func NewKV(config *api.Config) (*KV, error) {
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	kv := &KV{
		kv:           client.KV(),
		WriteOptions: &api.WriteOptions{},
		QueryOptions: &api.QueryOptions{},
		Prefix:       "",
	}

	return kv, nil
}

func (kv KV) Get(key string) (*api.KVPair, *api.QueryMeta, error) {
	return kv.kv.Get(key, kv.QueryOptions)
}

func (kv KV) List(key string) (api.KVPairs, *api.QueryMeta, error) {
	return kv.kv.List(key, kv.QueryOptions)
}

func (kv KV) Put(pair *api.KVPair) (*api.WriteMeta, error) {
	return kv.kv.Put(pair, kv.WriteOptions)
}

func (kv KV) Delete(key string) (*api.WriteMeta, error) {
	return kv.kv.Delete(key, kv.WriteOptions)
}
