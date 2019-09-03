package kapp

import (
	"k8s.io/client-go/kubernetes"
	"kappagent/kapp/v1"
	"kappagent/kapp/v2"
	"kappagent/util/k8s"
	"kappagent/util/tool"
	"regexp"
)

type Kapp struct {
	ClientSet *kubernetes.Clientset
	V1Agent v1.Service
	V2Agent v2.Service
}

func Run(clusterName string,cloud string,siteUrl string,regExp *regexp.Regexp){
	clientSet := k8s.InitClient()
	kapp := &Kapp{
		ClientSet: clientSet,
		V1Agent: v1.NewV1Agent(clientSet,clusterName,cloud,siteUrl,regExp),
		V2Agent: v2.NewV2Agent(clientSet,clusterName,cloud,siteUrl,regExp),
	}

	tool.Log.Infof("集群版本: %s",kapp.getVersion())
	if re, _ := regexp.Compile("1.7.8");re.MatchString(kapp.getVersion()){
		for {
			if success := kapp.V1Agent.StartRegCluster(); success {
				break
			}
		}
		go kapp.V1Agent.GetChannel()
		go kapp.V1Agent.StartWatchDeployment()
		go kapp.V1Agent.StartWatchStatefulSet()
		go kapp.V1Agent.StartWatchNode()
	}else{
		for {
			if success := kapp.V2Agent.StartRegCluster(); success {
				break
			}
		}
		go kapp.V2Agent.GetChannel()
		go kapp.V2Agent.StartWatchDeployment()
		go kapp.V2Agent.StartWatchStatefulSet()
		go kapp.V2Agent.StartWatchNode()
	}
	select {}
}

func (k *Kapp) getVersion() string{
	version,_ := k.ClientSet.ServerVersion()
	return version.String()
}