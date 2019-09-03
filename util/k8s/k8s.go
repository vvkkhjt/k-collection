package k8s

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kappagent/util/tool"
	"os"
	"path/filepath"
)

const (
	envRunEnv = "RUN_ENV"
)

var (
	runEnv = "DEV"
)

func init(){
	if re := os.Getenv(envRunEnv); re != "" {
		runEnv = os.Getenv(envRunEnv)
	}
}

// 初始化k8s client
func InitClient() *kubernetes.Clientset {
	tool.Log.Info("初始化client...")
	// 本地开发
	var kConfig *rest.Config
	if runEnv == "DEV" {
		var kubeConfig *string
		if home := tool.HomeDir(); home != "" {
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

	tool.Log.Info("初始化client成功...")

	return clientSet
}
