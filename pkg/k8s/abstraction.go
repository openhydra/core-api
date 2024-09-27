package k8s

import (
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var k8sHelper IK8sHelper

type K8sHelperType string

const (
	DefaultK8sHelperType K8sHelperType = "default"
	FakeK8sHelperType    K8sHelperType = "fake"
)

type IK8sHelper interface {
	GetConfigMapData(namespace, name string) (map[string]string, error)
	GetNodes() (coreV1.NodeList, error)
	GetAllPods() (coreV1.PodList, error)
	UpdateConfigMapData(namespace, name string, data map[string]string) error
}

// k8s helper is a singleton, it will be initialized only once
func InitK8sHelper(helpType K8sHelperType, clientSet *kubernetes.Clientset, stopChan <-chan struct{}) error {
	switch helpType {
	case DefaultK8sHelperType:
		if k8sHelper == nil {
			if clientSet == nil || stopChan == nil {
				return fmt.Errorf("either rest config or stopChan is nil")
			}
			defaultHelper := &DefaultK8sHelper{
				clientSet: clientSet,
			}
			defaultHelper.RunInformer(stopChan)
			k8sHelper = defaultHelper
		}
	case FakeK8sHelperType:
		// for fake for unit test, we will always create new instance
		k8sHelper = &Fake{}
	default:
		return fmt.Errorf("unknown k8s helper type: %s", helpType)
	}
	return nil
}

func GetK8sHelper() (IK8sHelper, error) {
	if k8sHelper == nil {
		return nil, fmt.Errorf("k8s helper is not initialized")
	}
	return k8sHelper, nil
}
