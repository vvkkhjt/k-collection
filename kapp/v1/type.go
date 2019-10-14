package v1

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	extensionsbeta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/watch"
)

// 老集群数据结构
type Project struct {
	ClusterName string      `json:"clusterName"`
	Timestamp   int64       `json:"timestamp"`
	Namespaces  []Namespace `json:"namespaces"`
	Nodes       []v1.Node   `json:"nodes"`
	Cloud       string      `json:"cloud"`
}

type Namespace struct {
	Name         string        `json:"name"`
	Deployments  []Deployment  `json:"deployments"`
	StatefulSets []StatefulSet `json:"statefulsets"`
}

type Deployment struct {
	Data extensionsbeta1.Deployment `json:"data"`
	Pods []Pod                      `json:"pods"`
}

type StatefulSet struct {
	Data v1beta1.StatefulSet `json:"data"`
	Pods []Pod               `json:"pods"`
}

type Pod struct {
	Data       v1.Pod      `json:"data"`
	Containers []Container `json:"containers"`
}

type Container struct {
	Data v1.Container `json:"data"`
}

type WatchProject struct {
	ClusterName  string          `json:"clusterName"`
	Timestamp    int64           `json:"timestamp"`
	ResourceType string          `json:"resourceType"`
	Type         watch.EventType `json:"type"`
	Namespaces   []Namespace     `json:"namespaces"`
}

type WatchDepData struct {
	Deployment *extensionsbeta1.Deployment
	Type       watch.EventType
	Namespace  string
}

type WatchStatefulData struct {
	StatefulSet *v1beta1.StatefulSet
	Type        watch.EventType
	Namespace   string
}

type WatchNode struct {
	ClusterName  string          `json:"clusterName"`
	Timestamp    int64           `json:"timestamp"`
	ResourceType string          `json:"resourceType"`
	Type         watch.EventType `json:"type"`
	Node         v1.Node         `json:"node"`
}

type WatchNodeData struct {
	Node *v1.Node
	Type watch.EventType
}
