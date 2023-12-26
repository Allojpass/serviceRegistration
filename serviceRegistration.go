package main

import (
	"context"
	"flag"
	"io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type serviceRegistrationInformation struct {
	SName string `json:"sName"`
	LSip  string `json:"lSip"`
}

func main() {

	var kubeconfig *string
	//读取默认目录下的config文件，也就是k8s中kubectl用来控制集群的那个文件
	//默认文件路径为/root/.kube/config(以root用户登录)
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	stopChS := make(chan struct{})
	defer close(stopChS)
	sharedInformers := informers.NewSharedInformerFactory(clientset, time.Minute)

	informerService := sharedInformers.Core().V1().Services().Informer()
	informerService.AddEventHandler(cache.ResourceEventHandlerFuncs{
		//新增服务的事件被触发则调用以下的代码
		AddFunc: func(obj interface{}) {
			mObj := obj.(v1.Object)
			log.Println("The name of this new service is ", mObj.GetName())
			//获取服务的详细信息，方便判断是否是NodePort类型，如果是NodePort类型则需要注册
			service, _ := clientset.CoreV1().Services(mObj.GetNamespace()).Get(context.TODO(), mObj.GetName(), v1.GetOptions{})
			if service.Spec.Type == "NodePort" {
				//获取node节点的信息列表
				nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
				if err != nil {
					panic(err.Error())
				}
				//这里用集群中任意节点的ip作为lsip,这里选择第一个work节点
				postParam := &serviceRegistrationInformation{SName: mObj.GetName(), LSip: nodes.Items[1].Annotations["flannel.alpha.coreos.com/public-ip"]}
				reqParam, err := json.Marshal(postParam)
				if err != nil {
					log.Println("Failed to convert json!")
				}
				log.Println(postParam)
				log.Println(string(reqParam))
				reqBody := strings.NewReader(string(reqParam))
				log.Println(reqBody)
				//注册服务信息
				resp, err := http.Post("http://127.0.0.1:5000/services/register", "application/json", reqBody)
				if err != nil {
					log.Println("Failed to register the service information!")
				} else {
					log.Println("Success to register the service information!")
					body, _ := io.ReadAll(resp.Body)
					bodystr := string(body)
					log.Println(bodystr)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			mObj := obj.(v1.Object)
			log.Println("The name of this deleted service is " + mObj.GetName())
			service, _ := clientset.CoreV1().Services(mObj.GetNamespace()).Get(context.TODO(), mObj.GetName(), v1.GetOptions{})
			if service.Spec.Type == "NodePort" {
				//获取node节点的信息列表
				nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
				if err != nil {
					panic(err.Error())
				}
				//这里用集群中任意节点的ip作为lsip,这里选择第一个work节点
				postParam := &serviceRegistrationInformation{SName: mObj.GetName(), LSip: nodes.Items[1].Annotations["flannel.alpha.coreos.com/public-ip"]}
				reqParam, err := json.Marshal(postParam)
				if err != nil {
					log.Println("Failed to convert json!")
				}
				log.Println(postParam)
				log.Println(string(reqParam))
				reqBody := strings.NewReader(string(reqParam))
				log.Println(reqBody)
				//注册服务信息
				resp, err := http.Post("http://127.0.0.1:5000/services/registercancel", "application/json", reqBody)
				if err != nil {
					log.Println("Failed to delete the service information!")
				} else {
					log.Println("Success to delete the service information!")
					body, _ := io.ReadAll(resp.Body)
					bodystr := string(body)
					log.Println(bodystr)
				}
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {

		},
	})
	informerService.Run(stopChS)
}
