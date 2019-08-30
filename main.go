/*
Copyright 2016 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"flag"
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kappagent/util"
	"os"
	"path/filepath"
	"time"
)

const (
	envClusterName = "CLUSTER_NAME"
	envSiteUrl     = "SITE_URL"
	envRunEnv      = "RUN_ENV"
	envCloud       = "CLOUD"
)

var (
	clusterName      = "build-cluster"
	cloud            = "inside"
	siteUrl          = "http://localhost:3000/cluster"
	//siteUrl          = "http://192.168.104.92:8000/api/k8s/k8sync/"
	runEnv           = "DEV"
	watchDeploymentChannel  = make(chan util.WatchDepData, 100)
	watchStatefulSetChannel  = make(chan util.WatchStatefulData, 100)
	watchNodeChannel = make(chan util.WatchNodeData, 100)
)

type WatchProject struct {
	ClusterName  string          `json:"clusterName"`
	Timestamp    int64           `json:"timestamp"`
	ResourceType string          `json:"resourceType"`
	Type         watch.EventType `json:"type"`
	Namespaces   []Namespace     `json:"namespaces"`
}

type WatchNode struct {
	ClusterName  string           `json:"clusterName"`
	Timestamp    int64            `json:"timestamp"`
	ResourceType string           `json:"resourceType"`
	Type         watch.EventType  `json:"type"`
	Node         v1.Node          `json:"node"`
}

type Project struct {
	ClusterName string           `json:"clusterName"`
	Timestamp   int64            `json:"timestamp"`
	Namespaces  []Namespace      `json:"namespaces"`
	Nodes       []v1.Node        `json:"nodes"`
	Cloud       string           `json:"cloud"`
}

type Namespace struct {
	Name        string       `json:"name"`
	Deployments []Deployment `json:"deployments"`
	StatefulSets []StatefulSet `json:"statefulsets"`
}

type Deployment struct {
	Data v1beta2.Deployment `json:"data"`
	Pods []Pod  `json:"pods"`
}

type StatefulSet struct {
	Data v1beta2.StatefulSet `json:"data"`
	Pods []Pod  `json:"pods"`
}

type Pod struct {
	Data       v1.Pod   `json:"data"`
	Containers []Container `json:"containers"`
}

type Container struct {
	Data	v1.Container `json:"data"`
}

func main() {
	if cn := os.Getenv(envClusterName); cn == "" {
		//panic("请填写集群名称")
		clusterName = "build-cluster"
	} else {
		clusterName = os.Getenv(envClusterName)
	}

	if cln := os.Getenv(envCloud); cln == "" {
		//panic("请填写集群名称")
		cloud = "inside"
	} else {
		cloud = os.Getenv(envCloud)
	}

	if su := os.Getenv(envSiteUrl); su == "" {
		//panic("请填写数据上报地址")
	} else {
		siteUrl = os.Getenv(envSiteUrl)
	}

	if re := os.Getenv(envRunEnv); re != "" {
		runEnv = os.Getenv(envRunEnv)
	}

	clientSet := initClient()

	for{
		if success := startRegCluster(clientSet); success{
			break
		}
	}

	go getChannel(clientSet)
	go startWatchDeployment(clientSet)
	go startWatchStatefulSet(clientSet)
	go startWatchNode(clientSet)

	select {}
}

// 初始化k8s client
func initClient() *kubernetes.Clientset {
	util.Log.Info("初始化client...")
	// 本地开发
	var kConfig *rest.Config
	if runEnv == "DEV" {
		var kubeConfig *string
		if home := util.HomeDir(); home != "" {
			kubeConfig = flag.String("kubeConfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeConfig file")
		} else {
			kubeConfig = flag.String("kubeConfig", "", "absolute path to the kubeConfig file")
		}
		flag.Parse()
		config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
		if err != nil {
			panic(err.Error())
		}
		kConfig = config
	} else {
		// 集群内部署
		config, err := rest.InClusterConfig()
		if err != nil {
			panic("初始化config失败:" + err.Error())
		}
		kConfig = config
	}

	// 获取client
	clientSet, err := kubernetes.NewForConfig(kConfig)
	if err != nil {
		panic("获取client失败:" + err.Error())
	}

	util.Log.Info("初始化client成功...")

	return clientSet
}

// 监听deployment变化
func startWatchDeployment(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			util.Log.Error(err)
		}
	}()

	for {
		if err := util.WatchDepHandler(clientSet, watchDeploymentChannel); err == nil {
			util.Log.Info("watch deployment is stop! restart now...")
		}
	}
}

// 监听statefulset变化
func startWatchStatefulSet(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			util.Log.Error(err)
		}
	}()

	for {
		if err := util.WatchStatefulHandler(clientSet,watchStatefulSetChannel); err == nil {
			util.Log.Info("watch statefulset is stop! restart now...")
		}
	}
}

// 监听node变化
func startWatchNode(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			util.Log.Error(err)
		}
	}()

	for {
		if err := util.WatchNodeHandler(clientSet,watchNodeChannel); err == nil {
			util.Log.Info("watch node is stop! restart now...")
		}
	}
}

// 注册cluster
func startRegCluster(clientSet *kubernetes.Clientset) bool {
	project := &Project{
		ClusterName: clusterName,
		Timestamp:   time.Now().Unix(),
		Namespaces:  getResourceWithNamespace(clientSet),
		Nodes:       getNode(clientSet),
		Cloud:       cloud,
	}

	jsonBytes, err := json.Marshal(project)
	if err != nil {
		util.Log.Error(err)
	}

	success := util.RegCluster(string(jsonBytes),siteUrl)
	return success
}

// 获取Resource
func getResourceWithNamespace(clientSet *kubernetes.Clientset) []Namespace {
	util.Log.Info("正在获取项目数据...")
	var ns []Namespace

	namespaceItems, _ := clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	nitems := namespaceItems.Items

	for i := range nitems {
		// 收集deployment
		nname := nitems[i].Name
		if nname == "default" || nname == "kube-system" || nname == "kube-public" ||
			nname == "local" || nname == "tools" || util.RegExp.MatchString(nname) {
			continue
		}

		deploymentsClient, _ := clientSet.AppsV1beta2().Deployments(nname).List(metav1.ListOptions{})
		ditems := deploymentsClient.Items
		var ds []Deployment
		var ss []StatefulSet

		if len(ditems) == 0 {
			util.Log.Infof("namespace: %s has no deployment", nname)
		} else {
			for q := range ditems {
				o := ditems[q]

				ps := getPod(clientSet, nname, o.Spec.Selector.MatchLabels)
				ds = append(ds, Deployment{Data: o, Pods: ps})
			}
		}

		// 收集statefulset
		statefulsetsClient,_ := clientSet.AppsV1beta2().StatefulSets(nname).List(metav1.ListOptions{})
		sitems := statefulsetsClient.Items
		if len(sitems) == 0{
			util.Log.Infof("namespace: %s has no statefulsets", nname)
		}else{
			for q := range sitems {
				o := sitems[q]

				ps := getPod(clientSet, nname, o.Spec.Selector.MatchLabels)
				ss = append(ss, StatefulSet{Data: o, Pods: ps})
			}
		}
		ns = append(ns, Namespace{Name: nname, Deployments: ds, StatefulSets: ss})
	}
	util.Log.Info("获取项目数据完成...")
	return ns
}

//获取pod和container
func getPod(clientSet *kubernetes.Clientset, namespace string, labelSelector map[string]string) []Pod {
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
func getNode(clientSet *kubernetes.Clientset) []v1.Node {
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

// 接收channel发送数据
func getChannel(clientSet *kubernetes.Clientset) {
	for {
		select {
		case e := <-watchDeploymentChannel:
			util.Log.Infof("%s Deployment,Name: %s,NameSpace: %s", e.Type, e.Deployment.Name, e.Namespace)
			watchProject := &WatchProject{
				ClusterName:  clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Deployment",
				Namespaces: []Namespace{
					{
						Name: e.Namespace,
						Deployments: []Deployment{
							{
								Data: *e.Deployment,
								Pods: getPod(clientSet, e.Namespace, e.Deployment.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				util.Log.Error(err)
			}

			util.HttpPostForm(string(jsonBytes),siteUrl)
		case e := <-watchStatefulSetChannel:
			util.Log.Infof("%s StatefulSet,Name: %s,NameSpace: %s", e.Type, e.StatefulSet.Name, e.Namespace)
			watchProject := &WatchProject{
				ClusterName:  clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "StatefulSet",
				Namespaces: []Namespace{
					{
						Name: e.Namespace,
						StatefulSets: []StatefulSet{
							{
								Data: *e.StatefulSet,
								Pods: getPod(clientSet, e.Namespace, e.StatefulSet.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				util.Log.Error(err)
			}

			util.HttpPostForm(string(jsonBytes),siteUrl)
		case e := <-watchNodeChannel:
			util.Log.Infof("%s Node,Addresses: %s", e.Type, e.Node.Status.Addresses)
			watchNode := &WatchNode{
				ClusterName:  clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Node",
				Node:         *e.Node,
			}

			jsonBytes, err := json.Marshal(watchNode)
			if err != nil {
				util.Log.Error(err)
			}

			util.HttpPostForm(string(jsonBytes),siteUrl)
		}
	}
}