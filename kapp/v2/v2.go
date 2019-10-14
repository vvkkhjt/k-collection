package v2

import (
	"encoding/json"
	"io"
	"k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"kappagent/util/tool"
	"regexp"
	"sync"
	"time"
)

type Agent struct {
	clientSet                    *kubernetes.Clientset
	clusterName                  string
	cloud                        string
	siteUrl                      string
	regExp                       *regexp.Regexp
	watchDeploymentChannel       chan WatchDepData
	watchStatefulSetChannel      chan WatchStatefulData
	watchNodeChannel             chan WatchNodeData
	mutex                        sync.RWMutex
	closed                       bool
	closeWatchChannel            chan int
	closeWatchDeploymentChannel  chan int
	closeWatchStatefulSetChannel chan int
	closeWatchNodeChannel        chan int
	closer                       sync.WaitGroup
}

type Service interface {
	StartRegCluster() bool
	Run()
	Close()
}

func NewV2Agent(clientSet *kubernetes.Clientset, clusterName string, cloud string, siteUrl string, regExp *regexp.Regexp) Service {
	return &Agent{
		clientSet:                    clientSet,
		clusterName:                  clusterName,
		cloud:                        cloud,
		siteUrl:                      siteUrl,
		regExp:                       regExp,
		watchDeploymentChannel:       make(chan WatchDepData, 100),
		watchStatefulSetChannel:      make(chan WatchStatefulData, 100),
		watchNodeChannel:             make(chan WatchNodeData, 100),
		closeWatchChannel:            make(chan int, 1),
		closeWatchDeploymentChannel:  make(chan int, 1),
		closeWatchStatefulSetChannel: make(chan int, 1),
		closeWatchNodeChannel:        make(chan int, 1),
	}
}

func (v2 *Agent) Run() {
	v2.closer.Add(4)
	go v2.startGetChannel()
	go v2.startWatchDeployment()
	go v2.startWatchStatefulSet()
	go v2.startWatchNode()
	v2.closer.Wait()
}

func (v2 *Agent) Close() {
	v2.closeWatchChannel <- 1
}

// 注册cluster
func (v2 *Agent) StartRegCluster() bool {
	project := &Project{
		ClusterName: v2.clusterName,
		Timestamp:   time.Now().Unix(),
		Namespaces:  v2.getResourceWithNamespace(),
		Nodes:       v2.getNode(),
		Cloud:       v2.cloud,
	}

	jsonBytes, err := json.Marshal(project)
	if err != nil {
		tool.Log.Error(err)
	}

	success := tool.RegCluster(string(jsonBytes), v2.siteUrl)
	return success
}

// 接收channel发送数据
func (v2 *Agent) startGetChannel() {
loop:
	for {
		select {
		case e := <-v2.watchDeploymentChannel:
			tool.Log.Infof("%s deployment,Name: %s,NameSpace: %s", e.Type, e.Deployment.Name, e.Namespace)
			watchProject := &WatchProject{
				ClusterName:  v2.clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Deployment",
				Namespaces: []Namespace{
					{
						Name: e.Namespace,
						Deployments: []Deployment{
							{
								Data: *e.Deployment,
								Pods: v2.getPod(e.Namespace, e.Deployment.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				tool.Log.Error(err)
			}

			tool.HttpPostForm(string(jsonBytes), v2.siteUrl, e.Type)
		case e := <-v2.watchStatefulSetChannel:
			tool.Log.Infof("%s statefulSet,Name: %s,NameSpace: %s", e.Type, e.StatefulSet.Name, e.Namespace)
			watchProject := &WatchProject{
				ClusterName:  v2.clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "StatefulSet",
				Namespaces: []Namespace{
					{
						Name: e.Namespace,
						StatefulSets: []StatefulSet{
							{
								Data: *e.StatefulSet,
								Pods: v2.getPod(e.Namespace, e.StatefulSet.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				tool.Log.Error(err)
			}

			tool.HttpPostForm(string(jsonBytes), v2.siteUrl, e.Type)
		case e := <-v2.watchNodeChannel:
			tool.Log.Infof("%s Node,Addresses: %s", e.Type, e.Node.Status.Addresses)
			watchNode := &WatchNode{
				ClusterName:  v2.clusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Node",
				Node:         *e.Node,
			}

			jsonBytes, err := json.Marshal(watchNode)
			if err != nil {
				tool.Log.Error(err)
			}

			tool.HttpPostForm(string(jsonBytes), v2.siteUrl, e.Type)
		case <-v2.closeWatchChannel:
			v2.mutex.Lock()

			if !v2.closed {
				v2.closed = true
				v2.closeWatchDeploymentChannel <- 1
				close(v2.watchDeploymentChannel)
				v2.closeWatchStatefulSetChannel <- 1
				close(v2.watchStatefulSetChannel)
				v2.closeWatchNodeChannel <- 1
				close(v2.watchNodeChannel)
			}

			v2.mutex.Unlock()
			tool.Log.Info("正在关闭数据发送通道")
			break loop
		}
	}
	v2.closer.Done()
}

// 监听资源变化
func (v2 *Agent) startWatchDeployment() {
	for {
		if err := v2.watchDepHandler(); err == nil {
			tool.Log.Info("watch deployment is stop! restart now...")
		} else {
			tool.Log.Info("正在关闭DeploymentWatch")
			break
		}
	}
	v2.closer.Done()
}

func (v2 *Agent) startWatchStatefulSet() {
	for {
		if err := v2.watchStatefulHandler(); err == nil {
			tool.Log.Info("watch statefulset is stop! restart now...")
		} else {
			tool.Log.Info("正在关闭StatefulSetWatch")
			break
		}
	}
	v2.closer.Done()
}

func (v2 *Agent) startWatchNode() {
	for {
		if err := v2.watchNodeHandler(); err == nil {
			tool.Log.Info("watch node is stop! restart now...")
		} else {
			tool.Log.Info("正在关闭NodeWatch")
			break
		}
	}
	v2.closer.Done()
}

// watch handler
func (v2 *Agent) watchDepHandler() error {
	tool.Log.Info("正在监听deployment...")
	deploymentsClient := v2.clientSet.AppsV1beta2().Deployments(metav1.NamespaceAll)

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
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!v2.regExp.MatchString(nname) {
						data := WatchDepData{
							Deployment: e.Object.(*v1beta2.Deployment),
							Namespace:  e.Object.(*v1beta2.Deployment).Namespace,
							Type:       e.Type,
						}
						v2.watchDeploymentChannel <- data
					}
				}
			}
		case <-v2.closeWatchDeploymentChannel:
			break loop
		}
	}

	if v2.closed {
		return io.ErrClosedPipe
	}
	return nil
}

func (v2 *Agent) watchStatefulHandler() error {
	tool.Log.Info("正在监听statefulset...")
	statefulSetClient := v2.clientSet.AppsV1beta2().StatefulSets(metav1.NamespaceAll)

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
					nname := e.Object.(*v1beta2.StatefulSet).Namespace
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!v2.regExp.MatchString(nname) {
						data := WatchStatefulData{
							StatefulSet: e.Object.(*v1beta2.StatefulSet),
							Namespace:   e.Object.(*v1beta2.StatefulSet).Namespace,
							Type:        e.Type,
						}
						v2.watchStatefulSetChannel <- data
					}
				}
			}
		case <-v2.closeWatchStatefulSetChannel:
			break loop
		}
	}

	if v2.closed {
		return io.ErrClosedPipe
	}
	return nil
}

func (v2 *Agent) watchNodeHandler() error {
	tool.Log.Info("正在监听node...")
	nodesClient := v2.clientSet.CoreV1().Nodes()

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
					v2.watchNodeChannel <- data
				}
			}
		case <-v2.closeWatchNodeChannel:
			break loop
		}
	}

	if v2.closed {
		return io.ErrClosedPipe
	}
	return nil
}

// 获取Resource
func (v2 *Agent) getResourceWithNamespace() []Namespace {
	tool.Log.Info("正在获取项目数据...")
	var ns []Namespace

	namespaceItems, _ := v2.clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	nitems := namespaceItems.Items

	for i := range nitems {
		// 收集deployment
		nname := nitems[i].Name
		if nname == "default" || nname == "kube-system" || nname == "kube-public" ||
			nname == "local" || nname == "tools" || v2.regExp.MatchString(nname) {
			continue
		}
		var ss []StatefulSet
		var ds []Deployment

		deploymentsClient, _ := v2.clientSet.AppsV1beta2().Deployments(nname).List(metav1.ListOptions{})
		ditems := deploymentsClient.Items

		if len(ditems) == 0 {
			tool.Log.Infof("namespace: %s has no deployment", nname)
		} else {
			for q := range ditems {
				o := ditems[q]

				ps := v2.getPod(nname, o.Spec.Selector.MatchLabels)
				ds = append(ds, Deployment{Data: o, Pods: ps})
			}
		}

		// 收集statefulset
		statefulsetsClient, _ := v2.clientSet.AppsV1beta2().StatefulSets(nname).List(metav1.ListOptions{})
		sitems := statefulsetsClient.Items
		if len(sitems) == 0 {
			tool.Log.Infof("namespace: %s has no statefulsets", nname)
		} else {
			for q := range sitems {
				o := sitems[q]

				ps := v2.getPod(nname, o.Spec.Selector.MatchLabels)
				ss = append(ss, StatefulSet{Data: o, Pods: ps})
			}
		}

		ns = append(ns, Namespace{Name: nname, Deployments: ds, StatefulSets: ss})
	}
	tool.Log.Info("获取项目数据完成...")
	return ns
}

func (v2 *Agent) getPod(namespace string, labelSelector map[string]string) []Pod {
	pods, _ := v2.clientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{
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

func (v2 *Agent) getNode() []corev1.Node {
	tool.Log.Info("正在获取Node数据...")
	var nodes []corev1.Node

	nodesClient := v2.clientSet.CoreV1().Nodes()
	list, _ := nodesClient.List(metav1.ListOptions{})
	items := list.Items

	for _, v := range items {
		nodes = append(nodes, v)
	}
	tool.Log.Info("获取Node数据完成...")
	return nodes
}
