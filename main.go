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
	clusterName = "default-cluster"
	cloud       = "default-cloud"
	//siteUrl     = "http://localhost:3000/cluster"
	siteUrl          = "http://192.168.104.73:8000/api/k8s/k8sync/"
	//siteUrl          = "http://192.168.220.70:30626/api/k8s/k8sync/"
	regExp, _ = regexp.Compile("^(c|p|u|user|cattle)-")
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

	ks := kapp.NewKapp(clusterName, cloud, siteUrl, regExp)
	//signalChan := make(chan os.Signal, 1)
	//signal.Notify(signalChan,
	//	os.Kill,
	//	os.Interrupt,
	//	syscall.SIGHUP,
	//	syscall.SIGINT,
	//	syscall.SIGTERM,
	//	syscall.SIGQUIT)

	//go func() {
	//	select{
	//		case <- signalChan:
	//			ks.Close()
	//	}
	//}()

	ks.Run()

	defer ks.Close()
	//defer tool.Log.Fatal(tool.KafkaWriter.Close())
}
