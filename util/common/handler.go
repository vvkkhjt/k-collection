package common

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	. "kappagent/global"
	"kappagent/util"
)

//获取pod和container
func GetPod(clientSet *kubernetes.Clientset, namespace string, labelSelector map[string]string) []Pod {
	pods, _ := clientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector).String(),
	})
	items := pods.Items
	var ps []Pod

	for i := range items {
		o := items[i]
		var cs []Container
		for q := range o.Spec.Containers {
			cs = append(cs, Container{Data: o.Spec.Containers[q]})
		}
		ps = append(ps, Pod{Data: o, Containers: cs})
	}
	return ps
}

//获取node
func GetNode(clientSet *kubernetes.Clientset) []v1.Node {
	util.Log.Info("正在获取Node数据...")
	var nodeAddress []v1.Node

	nodesClient := clientSet.CoreV1().Nodes()
	list, _ := nodesClient.List(metav1.ListOptions{})
	items := list.Items

	for _, v := range items {
		nodeAddress = append(nodeAddress, v)
	}
	util.Log.Info("获取Node数据完成...")
	return nodeAddress
}