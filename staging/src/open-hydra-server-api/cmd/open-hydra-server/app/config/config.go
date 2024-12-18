package config

import (
	"time"
)

const (
	defaultKubeConfigDir     = ".kube/config"
	DefaultLeaseDuration     = 30 * time.Second
	DefaultRenewDeadline     = 15 * time.Second
	DefaultRetryPeriod       = 5 * time.Second
	DefaultResourceName      = "open-hydra-api-leader-lock"
	DefaultResourceLock      = "endpointsleases"
	DefaultResourceNamespace = "default"
	defaultResyncPeriod      = 30 * time.Minute
	defaultNamespace         = "open-hydra"
)

type (
	OpenHydraServerConfig struct {
		// in case of we want to control total pod number in cluster
		// -1 not limit pod allocatable will be count by resource wise
		PodAllocatableLimit int64 `json:"pod_allocatable_limit,omitempty" yaml:"podAllocatableLimit,omitempty"`
		// default = 2000
		// note unit is m
		DefaultCpuPerDevice uint64 `json:"default_cpu_per_device,omitempty" yaml:"defaultCpuPerDevice,omitempty"`
		// default = 8192
		// note unit is Mi
		DefaultRamPerDevice uint64 `json:"default_ram_per_device,omitempty" yaml:"defaultRamPerDevice,omitempty"`
		// default = 0
		DefaultGpuPerDevice uint8 `json:"default_gpu_per_device,omitempty" yaml:"defaultGpuPerDevice,omitempty"`
		// default = "/open-hydra/public-dataset"
		// all dataset upload by user will be store in this path
		PublicDatasetBasePath string `json:"dataset_base_path,omitempty" yaml:"datasetBasePath,omitempty"`
		PublicCourseBasePath  string `json:"course_base_path,omitempty" yaml:"courseBasePath,omitempty"`
		PublicDatasetMaxSize  int64  `json:"dataset_max_size,omitempty" yaml:"datasetMaxSize,omitempty"`
		PublicCourseMaxSize   int64  `json:"course_max_size,omitempty" yaml:"courseMaxSize,omitempty"`
		// default = "hostpath", hostpath or nfs
		// hostpath: open-hydra-server will use hostpath to mount dataset most likely for aio server or test
		// nfs: open-hydra-server will use nfs to mount dataset most likely for production
		PublicDatasetVolumeType string `json:"dataset_volume_type,omitempty" yaml:"datasetVolumeType,omitempty"`
		// default = "/root/public-dataset"
		PublicDatasetStudentMountPath string `json:"dataset_student_mount_path,omitempty" yaml:"datasetStudentMountPath,omitempty"`
		PublicCourseStudentMountPath  string `json:"course_student_mount_path,omitempty" yaml:"courseStudentMountPath,omitempty"`
		// should be no default value but fill it in installation script, because it is a runtime value
		// if not set we won't be able to start gpu pod at all
		DefaultGpuDriver string `json:"default_gpu_driver,omitempty" yaml:"defaultGpuDriver,omitempty"`
		// gpu resource keys that predefine for open-hydra-server to discover gpu resource
		GpuResourceKeys         []string            `json:"gpu_resource_keys,omitempty" yaml:"gpuResourceKeys,omitempty"`
		ServerIP                string              `json:"server_ip" yaml:"serverIP"`
		EnableJupyterLabBaseURL bool                `json:"enable_jupyter_lab_base_url" yaml:"enableJupyterLabBaseURL"`
		ApplyPortNameForIngress map[string]string   `json:"apply_port_name_for_ingress,omitempty" yaml:"applyPortNameForIngress,omitempty"`
		MySqlConfig             *MySqlConfig        `json:"mysql_config,omitempty" yaml:"mysqlConfig,omitempty"`
		EtcdConfig              *EtcdConfig         `json:"etcd_config,omitempty" yaml:"etcdConfig,omitempty"`
		DBType                  string              `json:"db_type,omitempty" yaml:"dbType,omitempty"`
		DisableAuth             bool                `json:"disable_auth" yaml:"disableAuth"`
		PatchResourceNotRelease bool                `json:"patch_resource_not_release,omitempty" yaml:"patchResourceNotRelease,omitempty"`
		CpuOverCommitRate       uint8               `json:"cpu_over_commit_rate,omitempty" yaml:"cpuOverCommitRate,omitempty"`
		MemoryOverCommitRate    uint8               `json:"memory_over_commit_rate,omitempty" yaml:"memoryOverCommitRate,omitempty"`
		AuthDelegateConfig      *AuthDelegateConfig `json:"auth_delegate_config,omitempty" yaml:"authDelegateConfig,omitempty"`
		MaximumPortsPerSandbox  uint8               `json:"maximum_ports_per_sandbox,omitempty" yaml:"maximumPortsPerSandbox,omitempty"`
		WorkspacePath           string              `json:"workspace_path,omitempty" yaml:"workspacePath,omitempty"`
		KubeClientConfig        *KubeClientConfig   `json:"kube_client_config,omitempty" yaml:"kubeClientConfig,omitempty"`
	}
)

func DefaultConfig() *OpenHydraServerConfig {
	return &OpenHydraServerConfig{
		PodAllocatableLimit:           -1,
		DefaultCpuPerDevice:           2000,
		DefaultRamPerDevice:           8192,
		DefaultGpuPerDevice:           0,
		PublicDatasetBasePath:         "/mnt/public-dataset",
		PublicCourseBasePath:          "/mnt/public-course",
		PublicDatasetMaxSize:          1 << 30, // 1 GiB
		PublicCourseMaxSize:           1 << 30, // 1 GiB
		PublicDatasetVolumeType:       "hostpath",
		PublicDatasetStudentMountPath: "/root/notebook/dataset-public",
		PublicCourseStudentMountPath:  "/root/notebook/course-public",
		MySqlConfig:                   DefaultMySqlConfig(),
		EtcdConfig:                    DefaultEtcdConfig(),
		DefaultGpuDriver:              "nvidia.com/gpu",
		GpuResourceKeys:               []string{"nvidia.com/gpu", "amd.com/gpu"},
		ServerIP:                      "localhost",
		CpuOverCommitRate:             1, // no over commit for cpu by default,set to 2 cpu request will be divide by 2
		MemoryOverCommitRate:          1, // no over commit for memory by default,set to 2 meaning memory request will be divide by 2
		MaximumPortsPerSandbox:        3,
		WorkspacePath:                 "/mnt/workspace",
		KubeClientConfig:              &KubeClientConfig{QPS: 100, Burst: 200},
		ApplyPortNameForIngress:       map[string]string{"jupyter-lab": "lab"},
	}
}

type EtcdConfig struct {
	Endpoints []string `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	CAFile    string   `json:"ca_file,omitempty" yaml:"caFile,omitempty"`
	CertFile  string   `json:"cert_file,omitempty" yaml:"certFile,omitempty"`
	KeyFile   string   `json:"key_file,omitempty" yaml:"keyFile,omitempty"`
}

type MySqlConfig struct {
	Address      string `json:"address,omitempty" yaml:"address,omitempty"`
	Port         uint16 `json:"port,omitempty" yaml:"port,omitempty"`
	Username     string `json:"username,omitempty" yaml:"username,omitempty"`
	Password     string `json:"password,omitempty" yaml:"password,omitempty"`
	DataBaseName string `json:"database_name,omitempty" yaml:"databaseName,omitempty"`
	Protocol     string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Character    string `json:"character,omitempty" yaml:"character,omitempty"`
	Collation    string `json:"collation,omitempty" yaml:"collation,omitempty"`
}

type AuthDelegateConfig struct {
	// if KeystoneConfig is set to nil then auth plugin will fall backup to database auth
	KeystoneConfig *KeystoneConfig `json:"keystone_config,omitempty" yaml:"keystoneConfig,omitempty"`
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

type KubeClientConfig struct {
	QPS   float32 `json:"qps,omitempty" yaml:"qps,omitempty"`
	Burst int     `json:"burst,omitempty" yaml:"burst,omitempty"`
}

func DefaultEtcdConfig() *EtcdConfig {
	return &EtcdConfig{
		Endpoints: []string{"http://localhost:2379"},
		CAFile:    "",
		CertFile:  "",
		KeyFile:   "",
	}
}

func DefaultMySqlConfig() *MySqlConfig {
	return &MySqlConfig{
		Address:      "mysql.svc.cluster.local",
		Port:         3306,
		Username:     "root",
		Password:     "root",
		DataBaseName: "open-hydra",
		Protocol:     "tcp",
		Character:    "utf8mb3",
		Collation:    "utf8mb3_general_ci",
	}
}
