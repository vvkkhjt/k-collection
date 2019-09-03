package v1

import (
	"encoding/json"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsbeta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"kappagent/util/tool"
	"regexp"
	"time"
)

type Agent struct {
	clientSet               *kubernetes.Clientset
	clusterName             string
	cloud                   string
	siteUrl                 string
	regExp                  *regexp.Regexp
	watchDeploymentChannel  chan WatchDepData
	watchStatefulSetChannel chan WatchStatefulData
	watchNodeChannel        chan WatchNodeData
}

type Service interface {
	StartRegCluster() bool
	Run()
}

func NewV1Agent(clientSet *kubernetes.Clientset,clusterName string,cloud string,siteUrl string,regExp *regexp.Regexp) Service {
	return &Agent{
		clientSet:               clientSet,
		clusterName:             clusterName,
		cloud:                   cloud,
		siteUrl:                 siteUrl,
		regExp:                  regExp,
		watchDeploymentChannel:  make(chan WatchDepData, 100),
		watchStatefulSetChannel: make(chan WatchStatefulData, 100),
		watchNodeChannel:        make(chan WatchNodeData, 100),
	}
}

func (v1 *Agent) Run(){
	go v1.startGetChannel()
	go v1.startWatchDeployment()
	go v1.startWatchStatefulSet()
	go v1.startWatchNode()
	select {}
}
// 初始化注册集群
func (v1 *Agent) StartRegCluster() bool {
	project := &Project{
		ClusterName: v1.clusterName,
		Timestamp:   time.Now().Unix(),
		Namespaces:  v1.getResourceWithNamespace(),
		Nodes:       v1.getNode(),
		Cloud:       v1.cloud,
	}

	jsonBytes, err := json.Marshal(project)
	if err != nil {
		tool.Log.Error(err)
	}

	success := tool.RegCluster(string(jsonBytes), v1.siteUrl)
	return success
}

// 获取Resource
func (v1 *Agent) getResourceWithNamespace() []Namespace {
	tool.Log.Info("正在获取项目数据...")
	var ns []Namespace

	namespaceItems, _ := v1.clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	nitems := namespaceItems.Items

	for i := range nitems {
		// 收集deployment
		nname := nitems[i].Name
		if nname == "default" || nname == "kube-system" || nname == "kube-public" ||
			nname == "local" || nname == "tools" || v1.regExp.MatchString(nname) {
			continue
		}
		var ds []Deployment
		var ss []StatefulSet

		deploymentsClient, _ := v1.clientSet.ExtensionsV1beta1().Deployments(nname).List(metav1.ListOptions{})
		ditems := deploymentsClient.Items

		if len(ditems) == 0 {
			tool.Log.Infof("namespace: %s has no deployment", nname)
		} else {
			for q := range ditems {
				o := ditems[q]

				ps := v1.getPod(nname, o.Spec.Selector.MatchLabels)
				ds = append(ds, Deployment{Data: o, Pods: ps})
			}
		}

		// 收集statefulset
		statefulsetsClient, _ := v1.clientSet.AppsV1beta1().StatefulSets(nname).List(metav1.ListOptions{})
		sitems := statefulsetsClient.Items
		if len(sitems) == 0 {
			tool.Log.Infof("namespace: %s has no statefulsets", nname)
		} else {
			for q := range sitems {
				o := sitems[q]

				ps := v1.getPod(nname, o.Spec.Selector.MatchLabels)
				ss = append(ss, StatefulSet{Data: o, Pods: ps})
			}
		}

		ns = append(ns, Namespace{Name: nname, Deployments: ds, StatefulSets: ss})
	}
	tool.Log.Info("获取项目数据完成...")
	return ns
}

func (v1 *Agent) getPod(namespace string, labelSelector map[string]string) []Pod {
	pods, _ := v1.clientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{
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

func (v1 *Agent) getNode() []corev1.Node {
	tool.Log.Info("正在获取Node数据...")
	var nodes []corev1.Node

	nodesClient := v1.clientSet.CoreV1().Nodes()
	list, _ := nodesClient.List(metav1.ListOptions{})
	items := list.Items

	for _, v := range items {
		nodes = append(nodes, v)
	}
	tool.Log.Info("获取Node数据完成...")
	return nodes
}

// 接收channel发送数据
func (v1 *Agent) startGetChannel() {
	for {
		select {
		case e := <-v1.watchDeploymentChannel:
			tool.Log.Infof("%s Deployment,Name: %s,NameSpace: %s", e.Type, e.Deployment.Name, e.Namespace)
			watchProject := &WatchProject{
				ClusterName:  v1.clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Deployment",
				Namespaces: []Namespace{
					{
						Name: e.Namespace,
						Deployments: []Deployment{
							{
								Data: *e.Deployment,
								Pods: v1.getPod(e.Namespace, e.Deployment.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				tool.Log.Error(err)
			}

			tool.HttpPostForm(string(jsonBytes), v1.siteUrl)
		case e := <-v1.watchStatefulSetChannel:
			tool.Log.Infof("%s StatefulSet,Name: %s,NameSpace: %s", e.Type, e.StatefulSet.Name, e.Namespace)
			watchProject := &WatchProject{
				ClusterName:  v1.clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "StatefulSet",
				Namespaces: []Namespace{
					{
						Name: e.Namespace,
						StatefulSets: []StatefulSet{
							{
								Data: *e.StatefulSet,
								Pods: v1.getPod(e.Namespace, e.StatefulSet.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				tool.Log.Error(err)
			}

			tool.HttpPostForm(string(jsonBytes), v1.siteUrl)
		case e := <-v1.watchNodeChannel:
			tool.Log.Infof("%s Node,Addresses: %s", e.Type, e.Node.Status.Addresses)
			watchNode := &WatchNode{
				ClusterName:  v1.clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Node",
				Node:         *e.Node,
			}

			jsonBytes, err := json.Marshal(watchNode)
			if err != nil {
				tool.Log.Error(err)
			}

			tool.HttpPostForm(string(jsonBytes), v1.siteUrl)
		}
	}
}

// 监听资源变化
func (v1 *Agent) startWatchDeployment() {
	defer func() {
		err := recover()
		if err != nil {
			tool.Log.Error(err)
		}
	}()

	for {
		if err := v1.watchDepHandler(); err == nil {
			tool.Log.Info("watch deployment is stop! restart now...")
		}
	}
}

func (v1 *Agent) startWatchStatefulSet() {
	defer func() {
		err := recover()
		if err != nil {
			tool.Log.Error(err)
		}
	}()

	for {
		if err := v1.watchStatefulHandler(); err == nil {
			tool.Log.Info("watch statefulset is stop! restart now...")
		}
	}
}

func (v1 *Agent) startWatchNode() {
	defer func() {
		err := recover()
		if err != nil {
			tool.Log.Error(err)
		}
	}()

	for {
		if err := v1.watchNodeHandler(); err == nil {
			tool.Log.Info("watch node is stop! restart now...")
		}
	}
}

// watch handler
func (v1 *Agent) watchDepHandler() error {
	tool.Log.Info("正在监听deployment...")
	deploymentsClient := v1.clientSet.ExtensionsV1beta1().Deployments(metav1.NamespaceAll)

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
					nname := e.Object.(*extensionsbeta1.Deployment).Namespace
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!v1.regExp.MatchString(nname) {
						data := WatchDepData{
							Deployment: e.Object.(*extensionsbeta1.Deployment),
							Namespace:  e.Object.(*extensionsbeta1.Deployment).Namespace,
							Type:       e.Type,
						}
						v1.watchDeploymentChannel <- data
					}
				}
			}
		}
	}
	return nil
}

func (v1 *Agent) watchStatefulHandler() error {
	tool.Log.Info("正在监听statefulset...")
	statefulSetClient := v1.clientSet.AppsV1beta1().StatefulSets(metav1.NamespaceAll)

	list, _ := statefulSetClient.List(metav1.ListOptions{})
	items := list.Items

	timeoutSeconds := int64((15 * time.Minute).Seconds())
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, _ := statefulSetClient.Watch(options)
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
					nname := e.Object.(*v1beta1.StatefulSet).Namespace
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!v1.regExp.MatchString(nname) {
						data := WatchStatefulData{
							StatefulSet: e.Object.(*v1beta1.StatefulSet),
							Namespace:   e.Object.(*v1beta1.StatefulSet).Namespace,
							Type:        e.Type,
						}
						v1.watchStatefulSetChannel <- data
					}
				}
			}
		}
	}
	return nil
}

func (v1 *Agent) watchNodeHandler() error {
	tool.Log.Info("正在监听node...")
	nodesClient := v1.clientSet.CoreV1().Nodes()

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
					data := WatchNodeData{
						Node: e.Object.(*corev1.Node),
						Type: e.Type,
					}
					v1.watchNodeChannel <- data
				}
			}
		}
	}
	return nil
}