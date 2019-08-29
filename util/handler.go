package util

import (
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"regexp"
	"time"
)

var(
	RegExp,_ = regexp.Compile("^(c|p|u|user|cattle)-")
)

type WatchDepData struct {
	Deployment *v1beta2.Deployment
	Type       watch.EventType
	Namespace  string
}

type WatchStatefulData struct {
	StatefulSet *v1beta2.StatefulSet
	Type        watch.EventType
	Namespace   string
}

type WatchNodeData struct {
	Addresses []v1.NodeAddress
	Type      watch.EventType
}

func WatchDepHandler(clientSet *kubernetes.Clientset, watchDeploymentChannel chan WatchDepData) error {
	Log.Info("正在监听deployment...")
	deploymentsClient := clientSet.AppsV1beta2().Deployments(metav1.NamespaceAll)

	list, _ := deploymentsClient.List(metav1.ListOptions{})
	items := list.Items

	timeoutSeconds := int64((15 * time.Minute).Seconds())
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, _ := deploymentsClient.Watch(options)
	defer w.Stop()

	// 为了第一次不发送数据，启动watch第一次会输出所有的数据
	count := 0
	// watch有超时时间，如果不在listoption里面设置TimeoutSeconds，默认30到60分钟会断开链接，
	// 所以用ok来监视是否断开链接
loop:
	for {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				break loop
			} else if e.Type == watch.Added || e.Type == watch.Deleted || e.Type == watch.Modified {
				if count != len(items) {
					count += 1
				} else {
					// go的断言获取运行时的struct
					nname := e.Object.(*v1beta2.Deployment).Namespace
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!RegExp.MatchString(nname) {
						data := WatchDepData{
							Deployment: e.Object.(*v1beta2.Deployment),
							Namespace:  e.Object.(*v1beta2.Deployment).Namespace,
							Type:       e.Type,
						}
						watchDeploymentChannel <- data
					}
				}
			}
		}
	}
	return nil
}

func WatchStatefulHandler(clientSet *kubernetes.Clientset, watchStatefulSetChannel chan WatchStatefulData) error {
	Log.Info("正在监听statefulset...")
	statefulSetClient := clientSet.AppsV1beta2().StatefulSets(metav1.NamespaceAll)

	list, _ := statefulSetClient.List(metav1.ListOptions{})
	items := list.Items

	timeoutSeconds := int64((15 * time.Minute).Seconds())
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, _ := statefulSetClient.Watch(options)
	defer w.Stop()

	// 为了第一次不发送数据，启动watch第一次会输出所有的数据
	count := 0
	// watch有超时时间，如果不在listoption里面设置TimeoutSeconds，默认30到60分钟会断开链接，
	// 所以用ok来监视是否断开链接
loop:
	for {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				break loop
			} else if e.Type == watch.Added || e.Type == watch.Deleted || e.Type == watch.Modified {
				if count != len(items) {
					count += 1
				} else {
					// go的断言获取运行时的struct
					nname := e.Object.(*v1beta2.StatefulSet).Namespace
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!RegExp.MatchString(nname) {
						data := WatchStatefulData{
							StatefulSet: e.Object.(*v1beta2.StatefulSet),
							Namespace:   e.Object.(*v1beta2.StatefulSet).Namespace,
							Type:        e.Type,
						}
						watchStatefulSetChannel <- data
					}
				}
			}
		}
	}
	return nil
}

func WatchNodeHandler(clientSet *kubernetes.Clientset, watchNodeChannel chan WatchNodeData) error {
	Log.Info("正在监听node...")
	nodesClient := clientSet.CoreV1().Nodes()

	list, _ := nodesClient.List(metav1.ListOptions{})
	items := list.Items

	timeoutSeconds := int64((15 * time.Minute).Seconds())
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, _ := nodesClient.Watch(options)
	defer w.Stop()

	// 为了第一次不发送数据，启动watch第一次会输出所有的数据
	count := 0
	// watch有超时时间，如果不在listoption里面设置TimeoutSeconds，默认30到60分钟会断开链接，
	// 所以用ok来监视是否断开链接
loop:
	for {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				break loop
			} else if e.Type == watch.Added || e.Type == watch.Deleted {
				if count != len(items) {
					count += 1
				} else {
					var addresses []v1.NodeAddress
					for _, v := range e.Object.(*v1.Node).Status.Addresses {
						addresses = append(addresses, v1.NodeAddress{
							Address: v.Address,
							Type:    v.Type,
						})
					}
					data := WatchNodeData{
						Addresses: addresses,
						Type:      e.Type,
					}
					watchNodeChannel <- data
				}
			}
		}
	}
	return nil
}
