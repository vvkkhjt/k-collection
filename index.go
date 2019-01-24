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

// Note: the example only works with the code within the same release/branch.
package main

import (
	"encoding/json"
	"errors"
	//"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
	"os"
	//"path/filepath"
	"regexp"
	"time"
)

const(
	envClusterName = "CLUSTER_NAME"
	envSiteUrl = "SITE_URL"
)

var (
	clusterName = "default-cluster"
	siteUrl = "http://140.143.83.18/cluster"
	sendChannel = make(chan int)
	watchChannel = make(chan map[string]interface{},100)
)

type Project struct {
	ClusterName string `json:"cluster_name"`
	Timestamp int64 `json:"timestamp"`
	Namespaces []Namespace `json:"namespaces"`
}

type Namespace struct {
	Name        string       `json:"name"`
	Deployments []Deployment `json:"deployments"`
}

type Deployment struct {
	Name string   `json:"name"`
	Pods []Pod `json:"pods"`
}

type Pod struct {
	Name string `json:"name"`
	Containers []string `json:"containers"`
}

func main() {
	if cn := os.Getenv(envClusterName);cn == ""{
		//panic("请填写集群名称")
		clusterName = "default-cluster"
	}else{
		clusterName = os.Getenv(envClusterName)
	}

	if su := os.Getenv(envSiteUrl);su == ""{
		panic("请填写数据上报地址")
	}else{
		siteUrl = os.Getenv(envSiteUrl)
	}

	clientSet := initClient()

	//go startWatchDp(clientSet)
	go getChannel(clientSet)
	go startWatchDeployment(clientSet)
	startGetProject()
	select {}
}
// 初始化k8s client
func initClient() *kubernetes.Clientset {
	log.Info("初始化client...")
	// 本地开发
	//var kubeConfig *string
	//if home := homeDir(); home != "" {
	//	kubeConfig = flag.String("kubeConfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeConfig file")
	//} else {
	//	kubeConfig = flag.String("kubeConfig", "", "absolute path to the kubeConfig file")
	//}
	//flag.Parse()
	//
	//config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	//if err != nil {
	//	panic(err.Error())
	//}

	// 集群内部署
	config, err := rest.InClusterConfig()
	if err != nil {
		panic("初始化config失败:" + err.Error())
	}

	// 获取client
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("获取client失败:" + err.Error())
	}

	log.Info("初始化client成功...")

	return clientSet
}

// 监听deployment变化
func startWatchDeployment(clientSet *kubernetes.Clientset){
	defer func() {
		err := recover()
		if err != nil {
			fmt.Println(err)
		}
	}()

	log.Info("正在监听deployment...")
	count := 0
	deploymentsClient := clientSet.AppsV1beta1().Deployments(metav1.NamespaceAll)
	list,_ := deploymentsClient.List(metav1.ListOptions{})
	items := list.Items
	w, _ := deploymentsClient.Watch(metav1.ListOptions{})
	for {
		select {
		case e, _ := <-w.ResultChan():
			if e.Type == watch.Added || e.Type == watch.Deleted || e.Type == watch.Modified{
				if count != len(items){
					count += 1
				}else{
					nname := e.Object.(*v1beta1.Deployment).Namespace
					if r, _ := regexp.Compile("^(p|u|user)-");nname != "default" && nname != "cattle-system" &&
						nname != "kube-system" && nname != "dsky-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" && !r.MatchString(nname) {
						data := make(map[string]interface{},1)
						data["type"] = e.Type
						data["name"] = e.Object.(*v1beta1.Deployment).Name
						data["namespace"] = e.Object.(*v1beta1.Deployment).Namespace
						watchChannel <- data
					}
				}
			}
		}
	}
}

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
//				watchChannel <- 1
//				//fmt.Println(obj)
//			},
//			DeleteFunc: func(obj interface{}) {
//				watchChannel <- 1
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

// 开始获取project
func startGetProject(){
	//defer func() {
	//	err := recover()
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//}()

	// 循环查询项目
	//for{
		sendChannel <- 1
		//time.Sleep( 4 * time.Hour)
	//}
}

// 获取deployment
func getDeployment(clientSet *kubernetes.Clientset) []Namespace {
	log.Info("正在获取项目数据...")
	var ns []Namespace

	namespaceItems, _ := clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	nitems := namespaceItems.Items

	for i := range nitems {
		nname := nitems[i].Name
		if r, _ := regexp.Compile("^(p|u|user)-");nname == "default" || nname == "cattle-system" ||
			nname == "kube-system" || nname == "dsky-system" ||
			nname == "kube-public" || nname == "local" || nname == "tools" || r.MatchString(nname) {
			continue
		}

		deploymentsClient, _ := clientSet.AppsV1beta1().Deployments(nname).List(metav1.ListOptions{})
		ditems := deploymentsClient.Items
		var ds []Deployment

		if len(ditems) == 0 {
			log.Info("no deployment")
		} else {
			for q := range ditems{
				o := ditems[q]

				ps := getPod(clientSet, nname, o.Name)
				ds = append(ds, Deployment{Name: o.Name, Pods: ps})
			}
		}
		ns = append(ns, Namespace{Name: nname, Deployments: ds})

		// TODO: 添加statefulset finder
		//statefulsetsClient,_ := clientset.AppsV1beta1().StatefulSets(nname).List(metav1.ListOptions{})
		//sitems := statefulsetsClient.Items
		//if len(sitems) == 0{
		//	fmt.Println("no statefulsets")
		//}else{
		//	for q := 0;q<len(ditems);q++{
		//		fmt.Printf("statefulsets:%s\n",ditems[q].Name)
		//	}
		//}
	}
	log.Info("获取项目数据完成...")
	return ns
}

//获取pod和container
func getPod(clientSet *kubernetes.Clientset, namespace string, deploymentName string) []Pod {
	pods, _ := clientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	items := pods.Items
	var ps []Pod

	for i := range items{
		o := items[i]
		var cs []string
		if re, _ := regexp.Compile(deploymentName); re.MatchString(o.Name) {
			for q := range o.Spec.Containers{
				cs = append(cs,o.Spec.Containers[q].Name)
			}
			ps = append(ps, Pod{Name: o.Name,Containers:cs})
		}
	}
	return ps
}

// 格式化数据成json字符串
func formatJson(project *Project) string {
	//fmt.Println("正在格式化数据...")
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println("格式化数据完成...")
	return string(jsonBytes)
}

// 发送数据
func httpPostForm(data string) error {
	log.Info("正在发送数据...")
	resp, err := http.PostForm(siteUrl, url.Values{"data": {data}})
	if err != nil {
		return errors.New("链接地址失败..."+err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200{
		log.Info("数据发送完成...")
	}else{
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.New("读取数据失败..."+err.Error())
		}
		log.Warn("数据发送失败...",string(body))
	}
	return nil
}

// 接收channel发送数据
func getChannel(clientSet *kubernetes.Clientset){
	for{
		select {
			case <-sendChannel:
				log.Infof("%s Deployment","Init")
				err := httpPostForm(formatJson(&Project{ClusterName: clusterName,Timestamp: time.Now().Unix(),Namespaces: getDeployment(clientSet)}))
				if err != nil{
					fmt.Println(err)
				}
			case e := <-watchChannel:
				log.Infof("%s Deployment,Name: %s,NameSpace: %s",e["type"],e["name"],e["namespace"])
				err := httpPostForm(formatJson(&Project{ClusterName: clusterName,Timestamp: time.Now().Unix(),Namespaces: getDeployment(clientSet)}))
				if err != nil{
					fmt.Println(err)
				}
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
