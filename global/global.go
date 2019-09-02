package global

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	extensionsbeta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/watch"
	"regexp"
)

var (
	ClusterName = "ccs-bj-ops-service"
	Cloud       = "Qcloud"
	SiteUrl     = "http://localhost:3000/cluster"
	//siteUrl          = "http://192.168.104.92:8000/api/k8s/k8sync/"
	//siteUrl          = "http://192.168.220.70:30626/api/k8s/k8sync/"
	ClusterVersion string
	RegExp,_ = regexp.Compile("^(c|p|u|user|cattle)-")
)

type WatchNode struct {
	ClusterName  string          `json:"clusterName"`
	Timestamp    int64           `json:"timestamp"`
	ResourceType string          `json:"resourceType"`
	Type         watch.EventType `json:"type"`
	Node         v1.Node         `json:"node"`
}

//新集群数据结构
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

// 老集群数据结构
type ProjectOld struct {
	ClusterName string         `json:"clusterName"`
	Timestamp   int64          `json:"timestamp"`
	Namespaces  []NamespaceOld `json:"namespaces"`
	Nodes       []v1.Node      `json:"nodes"`
	Cloud       string         `json:"cloud"`
}

type NamespaceOld struct {
	Name         string          `json:"name"`
	Deployments  []DeploymentOld `json:"deployments"`
	StatefulSets []StatefulSetOld   `json:"statefulsets"`
}

type DeploymentOld struct {
	Data extensionsbeta1.Deployment `json:"data"`
	Pods []Pod                      `json:"pods"`
}

type StatefulSetOld struct {
	Data v1beta1.StatefulSet `json:"data"`
	Pods []Pod               `json:"pods"`
}

type WatchProjectOld struct {
	ClusterName  string          `json:"clusterName"`
	Timestamp    int64           `json:"timestamp"`
	ResourceType string          `json:"resourceType"`
	Type         watch.EventType `json:"type"`
	Namespaces   []NamespaceOld     `json:"namespaces"`
}

type WatchOldDepData struct {
	Deployment *extensionsbeta1.Deployment
	Type       watch.EventType
	Namespace  string
}

type WatchOldStatefulData struct {
	StatefulSet *v1beta1.StatefulSet
	Type        watch.EventType
	Namespace   string
}

// 通用数据结构
type Pod struct {
	Data       v1.Pod      `json:"data"`
	Containers []Container `json:"containers"`
}

type Container struct {
	Data v1.Container `json:"data"`
}

type WatchNodeData struct {
	Node *v1.Node
	Type watch.EventType
}
