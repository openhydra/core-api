package k8s

import (
	"context"
	"fmt"

	coreApiLog "core-api/pkg/logger"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type DefaultK8sHelper struct {
	clientSet         *kubernetes.Clientset
	configMapInformer cache.SharedIndexInformer
	nodeInformer      cache.SharedIndexInformer
	podInformer       cache.SharedIndexInformer
}

func (helper *DefaultK8sHelper) GetConfigMapData(namespace, name string) (map[string]string, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)

	cm, exists, err := helper.configMapInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("config map %s not found", key)
	}

	// Assert the object to *coreV1.ConfigMap
	cmAsserted, ok := cm.(*coreV1.ConfigMap)
	if !ok {
		return nil, fmt.Errorf("object is not a ConfigMap")
	}

	return cmAsserted.Data, nil
}

func (helper *DefaultK8sHelper) GetNodes() (coreV1.NodeList, error) {
	nodes := helper.nodeInformer.GetStore().List()

	var nodesList coreV1.NodeList
	for _, nodeInterface := range nodes {
		if node, ok := nodeInterface.(*coreV1.Node); ok {
			nodesList.Items = append(nodesList.Items, *node)
		} else {
			fmt.Println("Type assertion failed")
		}
	}

	return nodesList, nil
}

func (helper *DefaultK8sHelper) GetAllPods() (coreV1.PodList, error) {
	// get all pods
	var podsList coreV1.PodList
	pods := helper.podInformer.GetStore().List()

	for _, podInterface := range pods {
		if pod, ok := podInterface.(*coreV1.Pod); ok {
			podsList.Items = append(podsList.Items, *pod)
		} else {
			fmt.Println("Type assertion failed")
		}
	}
	return podsList, nil
}

func (helper *DefaultK8sHelper) UpdateConfigMapData(namespace, name string, data map[string]string) error {
	key := fmt.Sprintf("%s/%s", namespace, name)

	cm, exists, err := helper.configMapInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("config map %s not found", key)
	}

	// Assert the object to *coreV1.ConfigMap
	cmAsserted, ok := cm.(*coreV1.ConfigMap)
	if !ok {
		return fmt.Errorf("object is not a ConfigMap")
	}

	cmAsserted.Data = data

	_, err = helper.clientSet.CoreV1().ConfigMaps(namespace).Update(context.Background(), cmAsserted, metaV1.UpdateOptions{})

	// retry for error like object cannot fill full update request
	if err != nil {
		coreApiLog.Logger.Error("Error updating config map, attempt to renew object from apiserver directly")
		// get from api server
		cm, err := helper.clientSet.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metaV1.GetOptions{})
		if err != nil {
			return err
		}
		cm.Data = data
		coreApiLog.Logger.Debug("Attempt to re-updating config map %s/%s", "namespace", namespace, "name", name)
		_, err = helper.clientSet.CoreV1().ConfigMaps(namespace).Update(context.Background(), cm, metaV1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("re-updating config map %s/%s failed, nothing can be done any more: %v", namespace, name, err)
		}
	}

	return nil
}

func (helper *DefaultK8sHelper) RunInformer(stopChan <-chan struct{}) {
	coreApiLog.Logger.Debug("Initializing DefaultK8sHelper")
	factory := informers.NewSharedInformerFactory(helper.clientSet, 0)
	helper.configMapInformer = factory.Core().V1().ConfigMaps().Informer()
	go helper.configMapInformer.Run(stopChan)
	helper.nodeInformer = factory.Core().V1().Nodes().Informer()
	go helper.nodeInformer.Run(stopChan)
	helper.podInformer = factory.Core().V1().Pods().Informer()
	go helper.podInformer.Run(stopChan)
	if !cache.WaitForCacheSync(stopChan, helper.configMapInformer.HasSynced, helper.nodeInformer.HasSynced, helper.podInformer.HasSynced) {
		coreApiLog.Logger.Error("failed to sync informers")
	}
	coreApiLog.Logger.Debug("DefaultK8sHelper informers synced")
}
