package config

import (
	chatV1 "core-api/pkg/north/api/chat/core/v1"

	"k8s.io/client-go/rest"
)

type Config struct {
	AuthConfig    *AuthConfig    `json:"auth,omitempty" yaml:"auth,omitempty"`
	KubeConfig    *KubeConfig    `json:"kube_config,omitempty" yaml:"kubeConfig,omitempty"`
	CoreApiConfig *CoreApiConfig `json:"core_api,omitempty" yaml:"coreApi,omitempty"`
	RayLLM        *RayLLM        `json:"ray_llm,omitempty" yaml:"rayLLM,omitempty"`
	Rag           *Rag           `json:"rag,omitempty" yaml:"rag,omitempty"`
	XInference    *XInference    `json:"x_inference,omitempty" yaml:"xInference,omitempty"`
}

type KubeConfig struct {
	RestConfig *rest.Config
}

type CoreApiConfig struct {
	Port string `json:"port,omitempty" yaml:"port,omitempty"`
	// should not use in production
	DisableAuth    bool   `json:"disable_auth,omitempty" yaml:"disableAuth,omitempty"`
	LogLevel       string `json:"log_level,omitempty" yaml:"logLevel,omitempty"`
	ReleaseVersion string `json:"release_version,omitempty" yaml:"releaseVersion,omitempty"`
	GitVersion     string `json:"git_version,omitempty" yaml:"gitVersion,omitempty"`
}

type AuthConfig struct {
	Keystone *KeystoneConfig `json:"keystone,omitempty" yaml:"keystone,omitempty"`
}

type KeystoneConfig struct {
	Endpoint           string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Username           string `json:"username,omitempty" yaml:"username,omitempty"`
	Password           string `json:"password,omitempty" yaml:"password,omitempty"`
	DomainId           string `json:"domain_id,omitempty" yaml:"domainId,omitempty"`
	ProjectId          string `json:"project_id,omitempty" yaml:"projectId,omitempty"`
	TokenKeyInResponse string `json:"token_key_in_response,omitempty" yaml:"tokenKeyInResponse,omitempty"`
	TokenKeyInRequest  string `json:"token_key_in_request,omitempty" yaml:"tokenKeyInRequest,omitempty"`
}

type RayLLM struct {
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

type XInference struct {
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

type Rag struct {
	Endpoint                     string                             `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Model                        string                             `json:"model,omitempty" yaml:"model,omitempty"`
	FileChatPath                 string                             `json:"file_chat_path,omitempty" yaml:"fileChatPath,omitempty"`
	QuickFileChatPath            string                             `json:"quick_file_chat_path,omitempty" yaml:"quickFileChatPath,omitempty"`
	ChatQuickStarts              map[string][]chatV1.ChatQuickStart `json:"chat_quick_starts,omitempty" yaml:"chatQuickStarts,omitempty"`
	MaximumChatHistoryRecord     int                                `json:"maximum_chat_history_record,omitempty" yaml:"maximumChatHistoryRecord,omitempty"`
	MaximumFileChatHistoryRecord int                                `json:"maximum_file_chat_history_record,omitempty" yaml:"maximumFileChatHistoryRecord,omitempty"`
	MaximumKbChatHistoryRecord   int                                `json:"maximum_kb_chat_history_record,omitempty" yaml:"maximumKbChatHistoryRecord,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		AuthConfig: &AuthConfig{
			Keystone: &KeystoneConfig{
				Endpoint:           "http://localhost:5000",
				Username:           "admin",
				Password:           "password",
				DomainId:           "default",
				ProjectId:          "default",
				TokenKeyInResponse: "X-Subject-Token",
				TokenKeyInRequest:  "X-Auth-Token",
			},
		},
		CoreApiConfig: &CoreApiConfig{
			Port:           "8080",
			LogLevel:       "info",
			ReleaseVersion: "v0.0.1-debug",
			GitVersion:     "v0.0.1-debug",
		},
		RayLLM: &RayLLM{
			Endpoint: "http://localhost:8081",
		},
		XInference: &XInference{
			Endpoint: "http://localhost:8082",
		},
		Rag: &Rag{
			Endpoint:                     "http://localhost:8082",
			Model:                        "Qwen-7B-chat",
			FileChatPath:                 "/mnt/aitutor_data",
			QuickFileChatPath:            "/mnt/quick_file_chat",
			MaximumChatHistoryRecord:     10,
			MaximumKbChatHistoryRecord:   10,
			MaximumFileChatHistoryRecord: 5,
		},
	}
}
