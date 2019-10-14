package kapp

import (
	"k8s.io/client-go/kubernetes"
	"kappagent/kapp/v1"
	"kappagent/kapp/v2"
	"kappagent/util/k8s"
	"kappagent/util/tool"
	"os"
	"regexp"
)

type Kapp struct {
	clientSet *kubernetes.Clientset
	v1Agent   v1.Service
	v2Agent   v2.Service
}

type KappService interface {
	Run()
	Close()
}

func NewKapp(clusterName, cloud, siteUrl string, regExp *regexp.Regexp) KappService {
	clientSet := k8s.InitClient()
	return &Kapp{
		clientSet: clientSet,
		v1Agent:   v1.NewV1Agent(clientSet, clusterName, cloud, siteUrl, regExp),
		v2Agent:   v2.NewV2Agent(clientSet, clusterName, cloud, siteUrl, regExp),
	}
}

func (k *Kapp) Run() {
	tool.Log.Infof("集群版本: %s", k.getVersion())
	if k.isOldCluster() {
		for {
			if success := k.v1Agent.StartRegCluster(); success {
				break
			}
		}
		k.v1Agent.Run()
	} else {
		for {
			if success := k.v2Agent.StartRegCluster(); success {
				break
			}
		}
		k.v2Agent.Run()
	}
}

func (k *Kapp) Close() {
	if k.isOldCluster() {
		k.v1Agent.Close()
	} else {
		k.v2Agent.Close()
	}
}

func (k *Kapp) isOldCluster() bool {
	flag := true
	sr, err := k.clientSet.ServerPreferredResources()
	if err != nil{
		tool.Log.Error("获取集群资源列表失败...")
		os.Exit(1)
	}
	for _, i := range sr {
		if i.GroupVersion == "apps/v1beta2" {
			flag = false
			break
		}
	}

	return flag
}

func (k *Kapp) getVersion() string {
	version, err := k.clientSet.ServerVersion()
	if err != nil{
		tool.Log.Error("获取集群版本失败...")
		os.Exit(1)
	}
	return version.String()
}
