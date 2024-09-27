package k8s

import (
	"fmt"

	coreV1 "k8s.io/api/core/v1"
)

type Fake struct{}

var fakeConfigMapCollection map[string]map[string]string = map[string]map[string]string{
	"open-hydra/openhydra-plugin": {
		"plugins": `{
	"defaultSandbox": "test",
	"sandboxes":{
		"test": {
			"display_title": "test",
			"cpuImageName": "test",
			"gpuImageSet": {
				"nvidia.com/gpu": "nvidia-gpu-image",
				"amd.com/gpu": ""
			},
			"icon_name": "test1.png",
			"command": ["test"],
			"description": "test",
			"developmentInfo": ["test"],
			"status": "test",
			"ports": [
				8888
			],
			"volume_mounts": [
				{
					"name": "jupyter-lab",
					"mount_path": "/root/notebook",
					"source_path": "/mnt/jupyter-lab"
				},
				{
					"name": "public-dataset",
					"mount_path": "/root/notebook/dataset-public",
					"source_path": "/mnt/public-dataset"
				},
				{
					"name": "public-course",
					"mount_path": "/mnt/public-course",
					"source_path": "/mnt/public-course"
				}
			]
		},
		"jupyter-lab": {
			"display_title": "jupyter-lab",
			"cpuImageName": "jupyter-lab-test",
			"gpuImageSet": {
				"nvidia.com/gpu": "nvidia-gpu-image",
				"amd.com/gpu": ""
			},
			"icon_name": "test2.png",
			"command": ["jupyter-lab-test"],
			"description": "jupyter-lab-test",
			"developmentInfo": ["jupyter-lab-test"],
			"status": "running",
			"ports": [
				8888
			],
			"volume_mounts": [
				{
					"name": "jupyter-lab",
					"mount_path": "/root/notebook",
					"source_path": "/mnt/jupyter-lab"
				},
				{
					"name": "public-dataset",
					"mount_path": "/root/notebook/dataset-public",
					"source_path": "/mnt/public-dataset"
				},
				{
					"name": "public-course",
					"mount_path": "/mnt/public-course",
					"source_path": "/mnt/public-course"
				}
			]
		},
		"jupyter-lab-lot-ports": {
			"display_title": "jupyter-lab-lot-ports",
			"cpuImageName": "jupyter-lab-test",
			"gpuImageSet": {
				"nvidia.com/gpu": "nvidia-gpu-image",
				"amd.com/gpu": ""
			},
			"icon_name": "test3.png",
			"command": ["jupyter-lab-test"],
			"description": "jupyter-lab-test",
			"developmentInfo": ["jupyter-lab-test"],
			"status": "running",
			"ports": [
				8888,
				8889,
				8890,
				8891
			],
			"volume_mounts": [
				{
					"name": "jupyter-lab",
					"mount_path": "/root/notebook",
					"source_path": "/mnt/jupyter-lab"
				},
				{
					"name": "public-dataset",
					"mount_path": "/root/notebook/dataset-public",
					"source_path": "/mnt/public-dataset"
				},
				{
					"name": "public-course",
					"mount_path": "/mnt/public-course",
					"source_path": "/mnt/public-course"
				}
			]
		},
		"jupyter-lab-not-ports": {
			"display_title": "jupyter-lab-test",
			"cpuImageName": "jupyter-lab-test",
			"gpuImageSet": {
				"nvidia.com/gpu": "nvidia-gpu-image",
				"amd.com/gpu": ""
			},
			"icon_name": "test4.png",
			"command": ["jupyter-lab-test"],
			"description": "jupyter-lab-test",
			"developmentInfo": ["jupyter-lab-test"],
			"status": "running"
		}
	}}`,
	},
}

func (helper *Fake) GetNodes() (coreV1.NodeList, error) {
	return coreV1.NodeList{}, nil
}

func (helper *Fake) GetAllPods() (coreV1.PodList, error) {
	return coreV1.PodList{}, nil
}

func (helper *Fake) UpdateConfigMapData(namespace, name string, data map[string]string) error {
	return nil
}

func (helper *Fake) GetConfigMapData(namespace, name string) (map[string]string, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)

	_, exists := fakeConfigMapCollection[key]
	if !exists {
		return nil, fmt.Errorf("config map %s not found", key)
	}

	return fakeConfigMapCollection[key], nil
}
