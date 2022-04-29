package main

import (
	//"fmt"
	"flag"
	"context"
	//"time"
	"log"
	"bufio"
	"os"
	"strings"
	//"io/ioutil"
	"sync"
	//"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/api/core/v1"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/util/duration"
)

func getPodLogs(podName string, podInterface kv1.PodInterface, wg *sync.WaitGroup, outChan chan string){ //cancelCtx context.Context, 
	log.Printf("Now watching pod %s\n", podName) //just testing
	defer wg.Done()
	defer log.Printf("Finished watching pod %s\n", podName) //testing
	//creating file to write to
	file, err := os.Create("/tmp/"+podName+".txt")
	checkErr(err)
	defer file.Close()
	log.Printf("file named "+podName+".txt created!") //testing
	PodLogsConnection := podInterface.GetLogs(podName, &v1.PodLogOptions{
		Follow:    true,
		TailLines: &[]int64{int64(10)}[0],
	})
	LogStream, _ := PodLogsConnection.Stream(context.Background())
	defer LogStream.Close()
	reader := bufio.NewScanner(LogStream)
	var exitString, line string
	for {
		for reader.Scan() {
			select {
			case exitString = <- outChan: 
				if (exitString == podName){ 
					return
				}
			default:
				_, err = file.WriteString(line)
				checkErr(err)
				log.Printf("Pod: %s line: %v\n", podName, line)
			}
		}
	}
}

func checkErr(err error){
	if err != nil {
		log.Fatal(err)
	}
}

func main(){
	//creating in-cluster config
	config, err := rest.InClusterConfig()
	checkErr(err)
	//creating clientset
	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)

	ns:= "jesko-ns" //rewrite so it gets the namespace where it is..for test purpose leave it as it is
	//not really necessary?
	var label, field string
	flag.StringVar(&label, "l", "", "Label selector")
	flag.StringVar(&field, "f", "", "Field selector")
	listOptions := metav1.ListOptions {
		LabelSelector: label,
		FieldSelector: field,
	}
	//not really necessary?

	api := clientset.CoreV1()
	podInterface := api.Pods(ns)
	//cpodList, err := podInterface.List(context.Background(), listOptions)
	checkErr(err)
	ctx := context.Background()
	//cancelCtx, endGofuncs := context.WithCancel(ctx) //bring this back only if you need to kill all goroutines or if all go routines need to talk to each other with common shit
	watcher, err := podInterface.Watch(ctx, listOptions)
    checkErr(err)
    ch := watcher.ResultChan()
	var wg sync.WaitGroup
	outChan := make(chan string)

	for event := range ch {
        pod, err := event.Object.(*v1.Pod)
        if !err{log.Fatal("udefined")}
		switch event.Type {
			case watch.Added:
				if !strings.Contains(pod.Name, "watcher") { //testing the name of pod
					log.Printf("Pod named %s added!\n", pod.Name) //optional
					wg.Add(1)
					go getPodLogs(pod.Name, podInterface, &wg, outChan)
				}
			case watch.Deleted:
				outChan <- pod.Name
				log.Printf("Pod named %s deleted!\n", pod.Name) //optional
		}
	}
	wg.Wait()
	//endGofuncs() 	
}