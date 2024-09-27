package v1

import (
	openhydraConfig "open-hydra-server-api/cmd/open-hydra-server/app/config"
	settingV1 "open-hydra-server-api/pkg/apis/open-hydra-api/setting/core/v1"
	summaryV1 "open-hydra-server-api/pkg/apis/open-hydra-api/summary/core/v1"
)

type Summary struct {
	TotalGpuAllocated            int64                                  `json:"totalGpuAllocated,omitempty"`
	TotalGpuAllocatable          int64                                  `json:"totalGpuAllocatable,omitempty"`
	TotalCpuRequestAllocated     int64                                  `json:"totalCpuRequestAllocated,omitempty"`
	TotalCpuAllocatable          int64                                  `json:"totalCpuAllocatable,omitempty"`
	TotalCpuLimitAllocated       int64                                  `json:"totalCpuLimitAllocated,omitempty"`
	TotalCpuUnit                 string                                 `json:"totalCpuUnit,omitempty"`
	TotalRamRequestAllocated     int64                                  `json:"totalRamRequestAllocated,omitempty"`
	TotalRamLimitAllocated       int64                                  `json:"totalRamLimitAllocated,omitempty"`
	TotalRamAllocatable          int64                                  `json:"totalRamAllocatable,omitempty"`
	TotalRamUnit                 string                                 `json:"totalRamUnit,omitempty"`
	TotalNodes                   int                                    `json:"totalNodes,omitempty"`
	TotalNodesCanUse             int                                    `json:"totalNodesCanUse,omitempty"`
	GpuResourceSumUp             map[string]summaryV1.GpuResourceSumUp  `json:"gpuResourceSumUp,omitempty"`
	GpuResourceShare             map[string]int                         `json:"gpuResourceShare,omitempty"`
	Setting                      *settingV1.Setting                     `json:"setting,omitempty"`
	OpenHydraConfig              *openhydraConfig.OpenHydraServerConfig `json:"openHydraConfig,omitempty"`
	TotalCpuAllocatableRaft      int64                                  `json:"totalCpuAllocatableRaft,omitempty"`
	TotalRamAllocatableRaft      int64                                  `json:"totalRamAllocatableRaft,omitempty"`
	TotalCpuRequestAllocatedRaft int64                                  `json:"totalCpuRequestAllocatedRaft,omitempty"`
	TotalRamRequestAllocatedRaft int64                                  `json:"totalRamRequestAllocatedRaft,omitempty"`
	TotalCpuLimitAllocatedRaft   int64                                  `json:"totalCpuLimitAllocatedRaft,omitempty"`
	TotalRamLimitAllocatedRaft   int64                                  `json:"totalRamLimitAllocatedRaft,omitempty"`
}
