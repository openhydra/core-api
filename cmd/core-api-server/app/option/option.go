package option

import (
	"core-api/cmd/core-api-server/app/config"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
)

var serverConfig *config.Config

type Option struct {
	ConfigPath                  string `json:"config_path" yaml:"configPath"`
	DoNotInitK8sInClusterConfig bool   `json:"disable_init_cluster" yaml:"disableInitCluster"`
}

func (opt *Option) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opt.ConfigPath, "config", "c", "", "config file path")
	fs.BoolVarP(&opt.DoNotInitK8sInClusterConfig, "disable-in-cluster", "d", false, "do not init k8s in cluster config, use this flag if you want to debug api that not related to k8s")
}

func (opt *Option) GenerateConfig(loadKubeConfig bool) (*config.Config, error) {
	if serverConfig != nil {
		return serverConfig, nil
	}

	serverConfig = config.DefaultConfig()

	// read file and unmarshal to config
	file, err := os.ReadFile(opt.ConfigPath)
	if err != nil {
		return nil, err
	}

	// unmarshal to config
	err = yaml.Unmarshal(file, serverConfig)
	if err != nil {
		return nil, err
	}

	if CheckCoreApiConfig(serverConfig) != nil {
		return nil, err
	}

	if !loadKubeConfig {
		return serverConfig, nil
	}

	if serverConfig.KubeConfig == nil {
		serverConfig.KubeConfig = &config.KubeConfig{}
	}

	if !opt.DoNotInitK8sInClusterConfig {
		restConfig, err := BuildKubeConfig(serverConfig.KubeClientConfig)
		if err != nil {
			return nil, err
		}

		serverConfig.KubeConfig.RestConfig = restConfig
	}

	return serverConfig, nil
}

func CheckCoreApiConfig(config *config.Config) error {
	if config.CoreApiConfig.LogLevel != "info" && config.CoreApiConfig.LogLevel != "debug" && config.CoreApiConfig.LogLevel != "error" && config.CoreApiConfig.LogLevel != "warn" {
		return fmt.Errorf("invalid log level: %s, expect to be one of info, debug, error, warn", config.CoreApiConfig.LogLevel)
	}
	return nil
}

func BuildKubeConfig(clientConfig *config.KubeClientConfig) (*rest.Config, error) {
	// note for now we only support in-cluster config for following reasons:
	// 1. we are running in k8s cluster
	// 2. we are using service account to access k8s api server
	var result *rest.Config
	// go with in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return config, err
	}

	// set QPS and Burst
	if clientConfig != nil {
		config.QPS = clientConfig.QPS
		config.Burst = clientConfig.Burst
	}

	// now we should read ca.crt for our purpose
	caCert, err := os.ReadFile(config.CAFile)
	if err != nil {
		return config, err
	}
	config.CAData = caCert
	result = config
	return result, nil
}
