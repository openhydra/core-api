package route

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/k8s"
	summaryCoreV1 "core-api/pkg/north/api/summary/core/v1"
	"core-api/pkg/south"
	"encoding/json"
	"fmt"
	"strings"

	gpuV1 "core-api/pkg/north/api/gpu/core/v1"
	openhydraConfig "open-hydra-server-api/cmd/open-hydra-server/app/config"
	openhydraApi "open-hydra-server-api/pkg/open-hydra/apis"

	utilCommon "core-api/pkg/util/common"

	"gopkg.in/yaml.v3"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type OpenhydraSettingSection string

var southRagHandler *south.RAGSouthApiHandler

func getOrInitSouthRagHandler(config *config.Config, stopChan <-chan struct{}) *south.RAGSouthApiHandler {
	if southRagHandler == nil {
		southRagHandler = south.NewRAGSouthApiHandler(config, stopChan)
		go southRagHandler.RunBackgroundCache()
	}
	return southRagHandler
}

var (
	OpenhydraSettingSectionStorage         OpenhydraSettingSection = "storage"
	OpenhydraSettingSectionRuntimeResource OpenhydraSettingSection = "runtimeResource"
	OpenhydraSettingSectionServerIp        OpenhydraSettingSection = "serverIp"
	OpenhydraSettingSectionGPUType         OpenhydraSettingSection = "gpuType"
)

func GetNvidiaGpuShare() (int, error) {
	// get k8s helper
	k8sHelper, err := k8s.GetK8sHelper()
	if err != nil {
		return 0, err
	}

	share, err := k8sHelper.GetConfigMapData("gpu-operator", "time-slicing-config-all")
	if err != nil {
		return 0, err
	}

	gpuShareInfo := &gpuV1.NvidiaGpuShare{}

	err = yaml.Unmarshal([]byte(share["any"]), gpuShareInfo)
	if err != nil {
		return 0, err
	}

	// parse string to int
	if gpuShareInfo.Sharing != nil {
		if gpuShareInfo.Sharing.TimeSlicing != nil {
			if len(gpuShareInfo.Sharing.TimeSlicing.Resources) > 0 {
				return gpuShareInfo.Sharing.TimeSlicing.Resources[0].Replicas, nil
			}
		}
	}

	return 1, nil
}

func GetOpenhydraConfigMap() (*openhydraConfig.OpenHydraServerConfig, error) {
	// get k8s helper
	k8sHelper, err := k8s.GetK8sHelper()
	if err != nil {
		return nil, err
	}

	configMapData, err := k8sHelper.GetConfigMapData("open-hydra", "open-hydra-config")
	if err != nil {
		return nil, err
	}

	// note we have to fill up not nil properties in the struct
	// to avoid mis-config

	currentConfig := &openhydraConfig.OpenHydraServerConfig{}
	err = yaml.Unmarshal([]byte(configMapData["config.yaml"]), currentConfig)
	if err != nil {
		return nil, err
	}

	defaultConfig := openhydraConfig.DefaultConfig()
	if currentConfig.CpuOverCommitRate == 0 {
		currentConfig.CpuOverCommitRate = defaultConfig.CpuOverCommitRate
	}

	if currentConfig.MemoryOverCommitRate == 0 {
		currentConfig.MemoryOverCommitRate = defaultConfig.MemoryOverCommitRate
	}

	if currentConfig.DefaultCpuPerDevice == 0 {
		currentConfig.DefaultCpuPerDevice = defaultConfig.DefaultCpuPerDevice
	}

	if currentConfig.DefaultRamPerDevice == 0 {
		currentConfig.DefaultRamPerDevice = defaultConfig.DefaultRamPerDevice
	}

	return currentConfig, nil
}

func UpdateOpenhydraConfigMap(config *openhydraConfig.OpenHydraServerConfig) error {
	// get k8s helper
	k8sHelper, err := k8s.GetK8sHelper()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = k8sHelper.UpdateConfigMapData("open-hydra", "open-hydra-config", map[string]string{"config.yaml": string(data)})
	if err != nil {
		return err
	}

	return nil
}

func ValidateOpenhydraConfig(config *openhydraConfig.OpenHydraServerConfig, section OpenhydraSettingSection) error {
	var errMsgs []string
	switch section {
	case OpenhydraSettingSectionStorage:
		if !utilCommon.DirExists(config.PublicDatasetBasePath) {
			errMsgs = append(errMsgs, fmt.Sprintf("storage path %s not exists", config.PublicDatasetBasePath))
		}
		if !utilCommon.DirExists(config.PublicCourseBasePath) {
			errMsgs = append(errMsgs, fmt.Sprintf("storage path %s not exists", config.PublicCourseBasePath))
		}
		if !utilCommon.DirExists(config.WorkspacePath) {
			errMsgs = append(errMsgs, fmt.Sprintf("storage path %s not exists", config.WorkspacePath))
		}
	case OpenhydraSettingSectionRuntimeResource:
		if config.CpuOverCommitRate == 0 {
			errMsgs = append(errMsgs, "cpu over commit rate should be greater than 0")
		}
		if config.MemoryOverCommitRate == 0 {
			errMsgs = append(errMsgs, "memory over commit rate should be greater than 0")
		}
		if config.DefaultCpuPerDevice == 0 {
			errMsgs = append(errMsgs, "default cpu per device should be greater than 0")
		}
		if config.DefaultRamPerDevice == 0 {
			errMsgs = append(errMsgs, "default ram per device should be greater than 0")
		}
	case OpenhydraSettingSectionServerIp:
		if !utilCommon.IsValidIPv4(config.ServerIP) {
			errMsgs = append(errMsgs, "server ip is invalid")
		}
	case OpenhydraSettingSectionGPUType:
		break
	default:
		return fmt.Errorf("unknown section %s", section)
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf(strings.Join(errMsgs, "\n"))
	}
	return nil
}

func GetOpenhydraPlugin() (*openhydraApi.PluginList, error) {
	// get k8s helper
	// it's ok to pass nil because it
	k8sHelper, err := k8s.GetK8sHelper()
	if err != nil {
		return nil, err
	}

	plugin := &openhydraApi.PluginList{}
	// get plugin list from config map
	data, err := k8sHelper.GetConfigMapData("open-hydra", "openhydra-plugin")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(data["plugins"]), plugin)
	if err != nil {
		return nil, err
	}

	return plugin, nil
}

func GetSummary() (*summaryCoreV1.Summary, error) {
	// get k8s helper
	k8sHelper, err := k8s.GetK8sHelper()
	if err != nil {
		return nil, err
	}

	nodes, err := k8sHelper.GetNodes()
	if err != nil {
		return nil, err
	}

	nodeTotal, nodeCanUse, cpuAllocatable, ramAllocatable := calculateNodeCanUse(nodes)

	ramMi := ramAllocatable.Value() / 1024 / 1024

	summary := &summaryCoreV1.Summary{
		TotalNodes:          nodeTotal,
		TotalNodesCanUse:    nodeCanUse,
		TotalCpuAllocatable: cpuAllocatable.Value() * 1000,
		TotalRamAllocatable: ramMi,
		TotalRamUnit:        "Mi",
		TotalCpuUnit:        "m",
	}

	pods, err := k8sHelper.GetAllPods()
	if err != nil {
		return nil, err
	}

	reqCpu, limCpu, reqRam, limRam := calculatePodsResourcesUsage(pods)
	summary.TotalCpuRequestAllocated = reqCpu.Value() * 1000
	summary.TotalCpuLimitAllocated = limCpu.Value() * 1000
	summary.TotalRamRequestAllocated = reqRam.Value() / 1024 / 1024
	summary.TotalRamLimitAllocated = limRam.Value() / 1024 / 1024

	summary.TotalCpuAllocatableRaft = cpuAllocatable.Value()
	summary.TotalRamAllocatableRaft = ramAllocatable.Value() / 1024 / 1024 / 1024
	summary.TotalCpuRequestAllocatedRaft = reqCpu.Value()
	summary.TotalRamRequestAllocatedRaft = reqRam.Value() / 1024 / 1024 / 1024
	summary.TotalCpuLimitAllocatedRaft = limCpu.Value()
	summary.TotalRamLimitAllocatedRaft = limRam.Value() / 1024 / 1024 / 1024

	// it's ok to pass nil because
	return summary, nil
}

func calculatePodsResourcesUsage(podList coreV1.PodList) (reqCpu, limCpu, reqRam, limRam *resource.Quantity) {
	reqCpu = resource.NewQuantity(0, resource.DecimalSI)
	limCpu = resource.NewQuantity(0, resource.DecimalSI)
	reqRam = resource.NewQuantity(0, resource.BinarySI)
	limRam = resource.NewQuantity(0, resource.BinarySI)
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			reqCpu.Add(container.Resources.Requests[coreV1.ResourceCPU])
			limCpu.Add(container.Resources.Limits[coreV1.ResourceCPU])
			reqRam.Add(container.Resources.Requests[coreV1.ResourceMemory])
			limRam.Add(container.Resources.Limits[coreV1.ResourceMemory])
		}
	}
	return
}

func calculateNodeCanUse(nodes coreV1.NodeList) (int, int, *resource.Quantity, *resource.Quantity) {
	canUse := 0
	cpuAllocatable := resource.NewQuantity(0, resource.DecimalSI)
	ramAllocatable := resource.NewQuantity(0, resource.BinarySI)
	for _, node := range nodes.Items {
		// check scheduable
		if !node.Spec.Unschedulable {
			// check taint
			canUse++
		}
		cpuAllocatable.Add(node.Status.Allocatable[coreV1.ResourceCPU])
		ramAllocatable.Add(node.Status.Allocatable[coreV1.ResourceMemory])
	}
	return len(nodes.Items), canUse, cpuAllocatable, ramAllocatable
}
