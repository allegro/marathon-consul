package main

import (
	"fmt"
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
	Prefix       string
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

func (kv KV) ensurePrefix(key string) string {
	if kv.Prefix != "" {
		key = fmt.Sprintf("%s/%s", kv.Prefix, key)
	}
	return key
}

func (kv KV) Put(pair *api.KVPair) (*api.WriteMeta, error) {
	pair.Key = kv.ensurePrefix(pair.Key)
	return kv.kv.Put(pair, kv.WriteOptions)
}

func (kv KV) Delete(key string) (*api.WriteMeta, error) {
	return kv.kv.Delete(kv.ensurePrefix(key), kv.WriteOptions)
}
