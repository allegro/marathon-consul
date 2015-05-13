package main

import (
	"github.com/hashicorp/consul/api"
)

type Putter interface {
	Put(*api.KVPair) (*api.WriteMeta, error)
}

type Deleter interface {
	Delete(string) (*api.WriteMeta, error)
}

type PutDeleter interface {
	Putter
	Deleter
}

type KV struct {
	kv           *api.KV
	WriteOptions *api.WriteOptions
}

func NewKV(config *api.Config) (*KV, error) {
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &KV{
		kv:           client.KV(),
		WriteOptions: &api.WriteOptions{},
	}, nil
}

func (kv KV) Put(pair *api.KVPair) (*api.WriteMeta, error) {
	return kv.kv.Put(pair, kv.WriteOptions)
}

func (kv KV) Delete(key string) (*api.WriteMeta, error) {
	return kv.kv.Delete(key, kv.WriteOptions)
}
