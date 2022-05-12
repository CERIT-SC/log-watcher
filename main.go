/*========================
		Log-watcher
	author: Patrik Jesko
	last update: 12/05/22
==========================*/

package main

import (
	"fmt"
	"context"
	"time"
	"log"
	"bufio"
	"os"
	"regexp"
	//"strings"
	//"errors"
	"sync"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
//Getting pod logs
func getPodLogs(pod *v1.Pod, podInterface kv1.PodInterface, cancelCtx context.Context, wg *sync.WaitGroup){

	fmt.Printf("Now watching pod %s\n", pod.Name) //just testing
	defer wg.Done()

	//creating file to write to
	file, err := os.Create("/tmp/"+pod.Name+".txt")
	checkErr(err)
	defer file.Close()
	log.Printf("file named "+pod.Name+".txt created!") //testing
	PodLogsConnection := podInterface.GetLogs(pod.Name, &v1.PodLogOptions{
		Follow:    true,
		//TailLines: &[]int64{int64(10)}[0], //without taillines it should show logs from creation/sinceSeconds
	})
	LogStream, err := PodLogsConnection.Stream(context.Background())
	checkErr(err)
	defer LogStream.Close()
	defer fmt.Printf("Stoped watching %s\n", pod.Name) //very important to see
	scanner := bufio.NewScanner(LogStream)
	var line string
	
	for {
		for scanner.Scan() { //returns bool so if Scan is false (when scan stops)
			select {
			case <-cancelCtx.Done(): //dont know what to do here :/
				break //I had return here maybe that was the problem (dont know when the context starts or whatever)
			default:
				line = scanner.Text()
				_, err = file.WriteString(line)
				checkErr(err)
				//fmt.Printf("Pod: %s line: %v\n", podName, line)
			}
		//!!!
		if scanner.Err() != nil { //if pod doesnt exist end this goroutine
			fmt.Printf("%s\n",scanner.Err())
		}
		}
	}
}
//Checking for error
func checkErr(err error){

	if err != nil {
		log.Fatal(err)
	}
}


func isPodRunning(podInterface kv1.PodInterface , podName string, ctx context.Context) wait.ConditionFunc {

	return func() (bool, error) {

		pod, err := podInterface.Get(ctx, podName, metav1.GetOptions{}) //IncludeUninitialized: true
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, fmt.Errorf("pod ran to completion")
		}
		return false, nil
	}
}

func main(){
	//creating in-cluster config
	config, err := rest.InClusterConfig()
	checkErr(err)
	//creating clientset
	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)
	
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace") //read the namespace name
    checkErr(err)

	api := clientset.CoreV1()
	podInterface := api.Pods(string(ns))
	checkErr(err)
	ctx := context.Background()
	cancelCtx, endGofuncs := context.WithCancel(ctx) //bring this back only if you need to kill all goroutines or if all go routines need to talk to each other with common shit
	watcher, err := podInterface.Watch(ctx, metav1.ListOptions{})
    checkErr(err)
    ch := watcher.ResultChan()
	var wg sync.WaitGroup

	r, _ := regexp.Compile(os.Getenv("FILTER"))

	for event := range ch {
        pod, err := event.Object.(*v1.Pod)
        if !err{log.Fatal("udefined")} //fatal is risky..
		switch event.Type {
			case watch.Added:
				if r.MatchString(pod.Name) {
					fmt.Printf("Pod named %s added!\n", pod.Name) //optional
					wait.PollImmediateInfinite(time.Second, isPodRunning(podInterface, pod.Name, ctx))
					wg.Add(1)
					go getPodLogs(pod, podInterface, cancelCtx, &wg)
				}
			case watch.Deleted:
				fmt.Printf("Pod named %s deleted!\n", pod.Name) //optional
		}
	}
	wg.Wait()
	endGofuncs() 	
}