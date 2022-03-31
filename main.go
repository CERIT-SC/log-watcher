package main

import (
	//"fmt"
	"flag"
	"context"
	//"path/filepath" //just for testing
	//"os" //just for testing
	//"time"
	"log"

	"k8s.io/client-go/rest" //this is for cluster deployment
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/api/core/v1"
	//kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/util/duration"
)



func main(){
	//creating in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	
	//creating clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	ns := "jesko-ns"
	var label, field string
	flag.StringVar(&label, "l", "", "Label selector")
	flag.StringVar(&field, "f", "", "Field selector")
	listOptions := metav1.ListOptions {
		LabelSelector: label,
		FieldSelector: field,
	}
	api := clientset.CoreV1()
	podInterface := api.Pods(ns)
	//podList, err := podInterface.List(context.Background(), listOptions)
	//if err != nil{
	//	log.Fatal(err)
	//}

	watcher, err := podInterface.Watch(context.Background(), listOptions)
    if err != nil {
      log.Fatal(err)
    }
    ch := watcher.ResultChan()

	for event := range ch {
        pod, ok := event.Object.(*v1.Pod)
        if !ok {
            log.Fatal("unexpected type")
        }
		switch event.Type {
			case watch.Added:
				log.Printf("Pod named %s added!\n", pod.Name)
			case watch.Deleted:
				log.Printf("Pod named %s deleted!\n", pod.Name)
				
		}
	}	
}