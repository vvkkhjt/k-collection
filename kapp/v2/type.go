package v2

import (
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

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
	Data v1beta2.Deployment `json:"data"`
	Pods []Pod              `json:"pods"`
}

type StatefulSet struct {
	Data v1beta2.StatefulSet `json:"data"`
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
	Deployment *v1beta2.Deployment
	Type       watch.EventType
	Namespace  string
}

type WatchStatefulData struct {
	StatefulSet *v1beta2.StatefulSet
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
