package main

import (
	"encoding/json"
	"github.com/hashicorp/consul/api"
)

type Service struct {
	Timestamp  string `json:"timestamp"`
	SlaveID    string `json:"slaveId"`
	TaskID     string `json:"taskId"`
	TaskStatus string `json:"taskStatus"`
	AppID      string `json:"appId"`
	Host       string `json:"host"`
	Ports      []int  `json:"ports"`
	Version    string `json:"version"`
}

func ParseService(event []byte) (*Service, error) {
	svc := &Service{}
	err := json.Unmarshal(event, svc)
	return svc, err
}

func (svc *Service) KV() *api.KVPair {
	serialized, _ := json.Marshal(svc)

	return &api.KVPair{
		Key:   svc.TaskID,
		Value: serialized,
	}
}
