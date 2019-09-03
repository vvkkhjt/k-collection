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
	"kappagent/kapp"
	"os"
	"regexp"
)

const (
	envClusterName = "CLUSTER_NAME"
	envSiteUrl     = "SITE_URL"
	envCloud       = "CLOUD"
)

var (
	clusterName = "ccs-bj-ops-service"
	cloud       = "Qcloud"
	siteUrl     = "http://localhost:3000/cluster"
	//siteUrl          = "http://192.168.104.92:8000/api/k8s/k8sync/"
	//siteUrl          = "http://192.168.220.70:30626/api/k8s/k8sync/"
	regExp,_ = regexp.Compile("^(c|p|u|user|cattle)-")
)

func main() {
	if cn := os.Getenv(envClusterName); cn != "" {
		//panic("请填写集群名称")
		clusterName = os.Getenv(envClusterName)
	}

	if cln := os.Getenv(envCloud); cln != "" {
		//panic("请填写集群名称")
		cloud = os.Getenv(envCloud)
	}

	if su := os.Getenv(envSiteUrl); su != "" {
		//panic("请填写数据上报地址")
		siteUrl = os.Getenv(envSiteUrl)
	}

	kapp.Run(clusterName,cloud,siteUrl,regExp)
}