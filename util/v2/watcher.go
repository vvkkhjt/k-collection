package v2

import (
	"k8s.io/client-go/kubernetes"
	. "kappagent/global"
	"kappagent/util"
)

var (
	WatchDeploymentChannel  = make(chan WatchDepData, 100)
	WatchStatefulSetChannel = make(chan WatchStatefulData, 100)
	WatchNodeChannel        = make(chan WatchNodeData, 100)
)

// 监听deployment变化
func StartWatchDeployment(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			util.Log.Error(err)
		}
	}()

	for {
		if err := WatchDepHandler(clientSet, WatchDeploymentChannel); err == nil {
			util.Log.Info("watch deployment is stop! restart now...")
		}
	}
}

// 监听statefulset变化
func StartWatchStatefulSet(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			util.Log.Error(err)
		}
	}()

	for {
		if err := WatchStatefulHandler(clientSet, WatchStatefulSetChannel); err == nil {
			util.Log.Info("watch statefulset is stop! restart now...")
		}
	}
}

// 监听node变化
func StartWatchNode(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			util.Log.Error(err)
		}
	}()

	for {
		if err := WatchNodeHandler(clientSet, WatchNodeChannel); err == nil {
			util.Log.Info("watch node is stop! restart now...")
		}
	}
}
