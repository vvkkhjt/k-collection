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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	. "kappagent/global"
	"kappagent/util"
	handlercommon "kappagent/util/common"
	handlerv1 "kappagent/util/v1"
	handlerv2 "kappagent/util/v2"
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
	runEnv = "DEV"
)

func main() {
	if cn := os.Getenv(envClusterName); cn != "" {
		//panic("请填写集群名称")
		ClusterName = os.Getenv(envClusterName)
	}

	if cln := os.Getenv(envCloud); cln != "" {
		//panic("请填写集群名称")
		Cloud = os.Getenv(envCloud)
	}

	if su := os.Getenv(envSiteUrl); su != "" {
		//panic("请填写数据上报地址")
		SiteUrl = os.Getenv(envSiteUrl)
	}

	if re := os.Getenv(envRunEnv); re != "" {
		runEnv = os.Getenv(envRunEnv)
	}

	clientSet := initClient()

	version, _ := clientSet.ServerVersion()
	ClusterVersion = version.String()
	re, _ := regexp.Compile("1.7.8")

	if re.MatchString(ClusterVersion) {
		for {
			if success := startRegOldCluster(clientSet); success {
				break
			}
		}
		go handlerv1.GetChannel(clientSet)
		go handlerv1.StartWatchDeployment(clientSet)
		go handlerv1.StartWatchStatefulSet(clientSet)
		go handlerv1.StartWatchNode(clientSet)
	} else {
		for {
			if success := startRegCluster(clientSet); success {
				break
			}
		}
		go handlerv2.GetChannel(clientSet)
		go handlerv2.StartWatchDeployment(clientSet)
		go handlerv2.StartWatchStatefulSet(clientSet)
		go handlerv2.StartWatchNode(clientSet)
	}

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

// 注册cluster
func startRegCluster(clientSet *kubernetes.Clientset) bool {
	project := &Project{
		ClusterName: ClusterName,
		Timestamp:   time.Now().Unix(),
		Namespaces:  handlerv2.GetResourceWithNamespace(clientSet),
		Nodes:       handlercommon.GetNode(clientSet),
		Cloud:       Cloud,
	}

	jsonBytes, err := json.Marshal(project)
	if err != nil {
		util.Log.Error(err)
	}

	success := util.RegCluster(string(jsonBytes), SiteUrl)
	return success
}

func startRegOldCluster(clientSet *kubernetes.Clientset) bool {
	project := &ProjectOld{
		ClusterName: ClusterName,
		Timestamp:   time.Now().Unix(),
		Namespaces:  handlerv1.GetResourceWithNamespace(clientSet),
		Nodes:       handlercommon.GetNode(clientSet),
		Cloud:       Cloud,
	}

	jsonBytes, err := json.Marshal(project)
	if err != nil {
		util.Log.Error(err)
	}

	success := util.RegCluster(string(jsonBytes), SiteUrl)
	return success
}

