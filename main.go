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