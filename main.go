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
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	envClusterName = "CLUSTER_NAME"
	envSiteUrl     = "SITE_URL"
	envRunEnv      = "RUN_ENV"
	envCloud       = "CLOUD"
)

var (
	clusterName      = "default-cluster"
	cloud            = "default-cloud"
	siteUrl          = "http://140.143.83.18/cluster"
	runEnv           = "DEV"
	regChannel       = make(chan int, 1)
	watchDepChannel  = make(chan WatchDepData, 100)
	watchNodeChannel = make(chan WatchNodeData, 100)
	Log              *logrus.Logger
)

type WatchDepData struct {
	DeploymentName string
	Type           watch.EventType
	Namespace      string
}

type WatchNodeData struct {
	Addresses []v1.NodeAddress
	Type      watch.EventType
}

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
	Node         []v1.NodeAddress `json:"node"`
}

type Project struct {
	ClusterName string           `json:"clusterName"`
	Timestamp   int64            `json:"timestamp"`
	Namespaces  []Namespace      `json:"namespaces"`
	Nodes       []v1.NodeAddress `json:"nodes"`
	Cloud       string           `json:"cloud"`
}

type Namespace struct {
	Name        string       `json:"name"`
	Deployments []Deployment `json:"deployments"`
}

type Deployment struct {
	Name string `json:"name"`
	Pods []Pod  `json:"pods"`
}

type Pod struct {
	Name       string   `json:"name"`
	Containers []string `json:"containers"`
}

func main() {
	if cn := os.Getenv(envClusterName); cn == "" {
		//panic("请填写集群名称")
		clusterName = "default-cluster"
	} else {
		clusterName = os.Getenv(envClusterName)
	}

	if cln := os.Getenv(envCloud); cln == "" {
		//panic("请填写集群名称")
		cloud = "default-cloud"
	} else {
		cloud = os.Getenv(envCloud)
	}

	if su := os.Getenv(envSiteUrl); su == "" {
		panic("请填写数据上报地址")
	} else {
		siteUrl = os.Getenv(envSiteUrl)
	}

	if re := os.Getenv(envRunEnv); re != "" {
		runEnv = os.Getenv(envRunEnv)
	}

	Log = NewLogger()
	clientSet := initClient()

	//go startWatchDp(clientSet)
	go getChannel(clientSet)
	go startWatchDeployment(clientSet)
	go startWatchNode(clientSet)
	//go startWatchConfigMap(clientSet)
	go startGetProject()
	select {}
}

// 初始化k8s client
func initClient() *kubernetes.Clientset {
	Log.Info("初始化client...")
	// 本地开发
	var kConfig *rest.Config
	if runEnv == "DEV" {
		var kubeConfig *string
		if home := homeDir(); home != "" {
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

	Log.Info("初始化client成功...")

	return clientSet
}

// 监听deployment变化
func startWatchDeployment(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			Log.Error(err)
		}
	}()

	for {
		if err := watchDepHandler(clientSet); err == nil {
			Log.Info("watch deployment is stop! restart now...")
		}
	}
}

// 监听node变化
func startWatchNode(clientSet *kubernetes.Clientset) {
	defer func() {
		err := recover()
		if err != nil {
			Log.Error(err)
		}
	}()

	for {
		if err := watchNodeHandler(clientSet); err == nil {
			Log.Info("watch node is stop! restart now...")
		}
	}
}

func watchDepHandler(clientSet *kubernetes.Clientset) error {
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
					if r, _ := regexp.Compile("^(c|p|u|user)-"); nname != "default" && nname != "cattle-system" &&
						nname != "kube-system" && nname != "dsky-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" && !r.MatchString(nname) {
						data := WatchDepData{
							DeploymentName: e.Object.(*v1beta2.Deployment).Name,
							Namespace:      e.Object.(*v1beta2.Deployment).Namespace,
							Type:           e.Type,
						}
						watchDepChannel <- data
					}
				}
			}
		}
	}
	return nil
}

func watchNodeHandler(clientSet *kubernetes.Clientset) error {
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

// 开始获取project
func startGetProject() {
	defer func() {
		err := recover()
		if err != nil {
			Log.Error(err)
		}
	}()

	// 循环查询项目
	//for{
	//	regChannel <- 1
	//	time.Sleep( 10 * time.Second)
	//}
	regChannel <- 1
}

// 获取deployment
func getDeploymentWithNamespace(clientSet *kubernetes.Clientset) []Namespace {
	Log.Info("正在获取项目数据...")
	var ns []Namespace

	namespaceItems, _ := clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	nitems := namespaceItems.Items

	for i := range nitems {
		nname := nitems[i].Name
		if r, _ := regexp.Compile("^(c|p|u|user)-"); nname == "default" || nname == "cattle-system" ||
			nname == "kube-system" || nname == "dsky-system" ||
			nname == "kube-public" || nname == "local" || nname == "tools" || r.MatchString(nname) {
			continue
		}

		deploymentsClient, _ := clientSet.AppsV1beta2().Deployments(nname).List(metav1.ListOptions{})
		ditems := deploymentsClient.Items
		var ds []Deployment

		if len(ditems) == 0 {
			Log.Infof("namespace: %s has no deployment", nname)
		} else {
			for q := range ditems {
				o := ditems[q]

				ps := getPod(clientSet, nname, o.Name)
				ds = append(ds, Deployment{Name: o.Name, Pods: ps})
			}
		}
		ns = append(ns, Namespace{Name: nname, Deployments: ds})

		// TODO: 添加statefulset finder
		//statefulsetsClient,_ := clientset.AppsV1beta2().StatefulSets(nname).List(metav1.ListOptions{})
		//sitems := statefulsetsClient.Items
		//if len(sitems) == 0{
		//	fmt.Println("no statefulsets")
		//}else{
		//	for q := 0;q<len(ditems);q++{
		//		fmt.Printf("statefulsets:%s\n",ditems[q].Name)
		//	}
		//}
	}
	Log.Info("获取项目数据完成...")
	return ns
}

//获取pod和container
func getPod(clientSet *kubernetes.Clientset, namespace string, deploymentName string) []Pod {
	pods, _ := clientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	items := pods.Items
	var ps []Pod

	for i := range items {
		o := items[i]
		var cs []string
		if re, _ := regexp.Compile(deploymentName); re.MatchString(o.Name) {
			for q := range o.Spec.Containers {
				cs = append(cs, o.Spec.Containers[q].Name)
			}
			ps = append(ps, Pod{Name: o.Name, Containers: cs})
		}
	}
	return ps
}

//获取node
func getNode(clientSet *kubernetes.Clientset) []v1.NodeAddress {
	Log.Info("正在获取Node数据...")
	var nodeAddress []v1.NodeAddress

	nodesClient := clientSet.CoreV1().Nodes()
	list, _ := nodesClient.List(metav1.ListOptions{})
	items := list.Items
	for _, v := range items {
		for _, vv := range v.Status.Addresses {
			nodeAddress = append(nodeAddress, v1.NodeAddress{
				Type:    vv.Type,
				Address: vv.Address,
			})
		}
	}
	Log.Info("获取Node数据完成...")
	return nodeAddress
}

// 发送数据
func httpPostForm(data string, register bool) {
	if register{
		Log.Info("正在注册数据...")
	}else{
		Log.Info("正在发送数据...")
	}

	resp, err := http.PostForm(siteUrl, url.Values{"data": {data}})
	if err != nil {
		Log.Error("链接地址失败:" + err.Error())
		if register {
			regChannel <- 1
		}
	}else{
		defer func() {
			err := resp.Body.Close()
			if err != nil{
				Log.Error(err.Error())
			}
		}()

		if resp.StatusCode == 200 {
			if register {
				Log.Info("数据注册完成...")
			} else {
				Log.Info("数据发送完成...")
			}
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Log.Error("读取数据失败:" + err.Error())
			}else{
				if register {
					Log.Warn("数据注册失败:", string(body))
				} else {
					Log.Warn("数据发送失败:", string(body))
				}
			}

			if register{
				regChannel <- 1
			}
		}
	}
}

// 接收channel发送数据
func getChannel(clientSet *kubernetes.Clientset) {
	for {
		select {
		case <-regChannel:
			project := &Project{
				ClusterName: clusterName,
				Timestamp:   time.Now().Unix(),
				Namespaces:  getDeploymentWithNamespace(clientSet),
				Nodes:       getNode(clientSet),
				Cloud:       cloud,
			}

			jsonBytes, err := json.Marshal(project)
			if err != nil {
				Log.Error(err)
			}

			httpPostForm(string(jsonBytes), true)
		case e := <-watchDepChannel:
			Log.Infof("%s Deployment,Name: %s,NameSpace: %s", e.Type, e.DeploymentName, e.Namespace)
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
								Name: e.DeploymentName,
								Pods: getPod(clientSet, e.Namespace, e.DeploymentName),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				Log.Error(err)
			}

			httpPostForm(string(jsonBytes), false)
		case e := <-watchNodeChannel:
			Log.Infof("%s Node,Addresses: %s", e.Type, e.Addresses)
			watchNode := &WatchNode{
				ClusterName:  clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Node",
				Node:         e.Addresses,
			}

			jsonBytes, err := json.Marshal(watchNode)
			if err != nil {
				Log.Error(err)
			}

			httpPostForm(string(jsonBytes), false)
		}
	}
}

// 获取本地k8s配置文件路径
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func NewLogger() *logrus.Logger {
	if Log != nil {
		return Log
	}
	pathMap := lfshook.PathMap{
		logrus.InfoLevel:  "./log/info.log",
		logrus.ErrorLevel: "./log/error.log",
	}
	Log = logrus.New()
	Log.Hooks.Add(lfshook.NewHook(
		pathMap,
		&logrus.JSONFormatter{},
	))
	return Log
}

//func startWatchConfigMap(clientSet *kubernetes.Clientset){
//	defer func() {
//		err := recover()
//		if err != nil {
//			fmt.Println(err)
//		}
//	}()
//
//	Log.Info("正在监听configmap...")
//	count := 0
//	configMaps := clientSet.CoreV1().ConfigMaps(metav1.NamespaceAll)
//	list,_ := configMaps.List(metav1.ListOptions{})
//	items := list.Items
//	w, _ := configMaps.Watch(metav1.ListOptions{})
//	for {
//		select {
//			case e, _ := <-w.ResultChan():
//				if e.Type == watch.Added || e.Type == watch.Deleted{
//					if count != len(items){
//						count += 1
//					}else{
//						// go的reflect获取运行时的struct
//						nname := e.Object.(*v1.ConfigMap).Namespace
//						println(nname)
//						//if r, _ := regexp.Compile("^(c|p|u|user)-");nname != "default" && nname != "cattle-system" &&
//						//	nname != "kube-system" && nname != "dsky-system" &&
//						//	nname != "kube-public" && nname != "local" && nname != "tools" && !r.MatchString(nname) {
//						//	data := make(map[string]interface{},1)
//						//	data["type"] = e.Type
//						//	data["name"] = e.Object.(*v1beta2.Deployment).Name
//						//	data["namespace"] = e.Object.(*v1beta2.Deployment).Namespace
//						//	watchDepChannel <- data
//						//}
//					}
//				}
//		}
//	}
//}

//func startWatchDp(clientSet *kubernetes.Clientset){
//	watchlist := cache.NewListWatchFromClient(
//		clientSet.AppsV1().RESTClient(),
//		"deployments",
//		metav1.NamespaceAll,
//		fields.Everything())
//
//	_, controller := cache.NewInformer(
//		watchlist,
//		&v13.Deployment{},
//		time.Millisecond*100,
//		cache.ResourceEventHandlerFuncs{
//			AddFunc: func(obj interface{}) {
//				watchDepChannel <- 1
//				//fmt.Println(obj)
//			},
//			DeleteFunc: func(obj interface{}) {
//				watchDepChannel <- 1
//			},
//		},
//	)
//
//	stop := make(chan struct{})
//	go controller.Run(stop)
//
//	for {
//		time.Sleep(10 * time.Second)
//	}
//}
