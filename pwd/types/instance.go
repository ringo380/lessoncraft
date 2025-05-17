package types

import "context"

type LessonContext struct {
	LessonID  string `json:"lesson_id" bson:"lesson_id"`
	StepIndex int    `json:"step_index" bson:"step_index"`
	Completed bool   `json:"completed" bson:"completed"`
}

type Instance struct {
	Name        string          `json:"name" bson:"name"`
	LessonCtx   *LessonContext  `json:"lesson_ctx,omitempty" bson:"lesson_ctx,omitempty"`
	Image       string          `json:"image" bson:"image"`
	Hostname    string          `json:"hostname" bson:"hostname"`
	IP          string          `json:"ip" bson:"ip"`
	RoutableIP  string          `json:"routable_ip" bson:"routable_id"`
	ServerCert  []byte          `json:"server_cert" bson:"server_cert"`
	ServerKey   []byte          `json:"server_key" bson:"server_key"`
	CACert      []byte          `json:"ca_cert" bson:"ca_cert"`
	Cert        []byte          `json:"cert" bson:"cert"`
	Key         []byte          `json:"key" bson:"key"`
	Tls         bool            `json:"tls" bson:"tls"`
	SessionId   string          `json:"session_id" bson:"session_id"`
	ProxyHost   string          `json:"proxy_host" bson:"proxy_host"`
	SessionHost string          `json:"session_host" bson:"session_host"`
	Type        string          `json:"type" bson:"type"`
	WindowsId   string          `json:"-" bson:"windows_id"`
	ctx         context.Context `json:"-" bson:"-"`
}

type WindowsInstance struct {
	Id        string `bson:"id"`
	SessionId string `bson:"session_id"`
}

type InstanceConfig struct {
	ImageName      string
	Privileged     bool
	Hostname       string
	ServerCert     []byte
	ServerKey      []byte
	CACert         []byte
	Cert           []byte
	Key            []byte
	Tls            bool
	PlaygroundFQDN string
	Type           string
	DindVolumeSize string
	Envs           []string
	Networks       []string
	LessonCtx      *LessonContext

	// Resource limits
	MaxProcesses int64  // Maximum number of processes (default: 1000)
	MaxMemoryMB  int64  // Maximum memory in MB (default: from environment)
	StorageSize  string // Maximum storage size (default: from environment)
}
