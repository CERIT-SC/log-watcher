/*========================
		Log-watcher
	author: Patrik Jesko
	last update: 31/05/22
==========================*/

/*
Need fixing:
pod not writing ouput longer than 30s -> err //problem on kubernetes side I think
when container restarts (depends how much it takes to restart) -> reading the old log till the new conainer isnt ready
not tested with lots of pods
change the restart count (makes deployment little bit messy)
*/


package main

import (
	//"path/filepath"
	"context"
	"strings"
	"regexp"
	"bufio"
	"time"
	"fmt"
	"log"
	"os"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/api/core/v1"
)

//Checking for error
func checkErr(err error){
	
	if err != nil {
		log.Fatal(err)
	}
}

//testing pod state
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

//testing status
func GetPodStatus(podInterface kv1.PodInterface , podName string, ctx context.Context) bool {
	
	pod, err := podInterface.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return true
	}	
	if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded {
		return true
	}
	return false
}

//wait till pod is restarted
func waitRestarted(pod *v1.Pod, restartCount *int32){

	for {
		if pod.Status.ContainerStatuses[0].RestartCount > *restartCount {
			*restartCount = pod.Status.ContainerStatuses[0].RestartCount
			log.Println(*restartCount)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

//Getting pod logs
func getPodLogs(pod *v1.Pod, podInterface kv1.PodInterface, ctx context.Context){

	fmt.Printf("Now watching pod %s\n", pod.Name) //optional
	file, err := os.Create("/tmp/"+pod.Name+".txt")
	checkErr(err)
	defer file.Close()
	PodLogsConnection := podInterface.GetLogs(pod.Name, &v1.PodLogOptions{
		Follow:    true,
	})
	LogStream, err := PodLogsConnection.Stream(ctx)
	checkErr(err)
	defer LogStream.Close()
	defer fmt.Printf("Stoped watching %s\n", pod.Name) //very important to see so we know the go routine ended //optional
	scanner := bufio.NewScanner(LogStream)
	var line string	
	var restartCount int32 = pod.Status.ContainerStatuses[0].RestartCount

	for {
		for scanner.Scan() {
			line = scanner.Text()
			if strings.Contains(line, "unable to retrieve container logs for containerd:") || strings.Contains(line, "an error occurred when try to find container") { 
				//true if .Text() returns token with errors in condition
				return
			} //temporary solution but gets the job done
			_, err = file.WriteString(line)
			checkErr(err)
		}
		
		if GetPodStatus(podInterface, pod.Name, ctx) || scanner.Err() != nil{ 
			//true if deleted or finished(job) || scanner has error
			return
		}
		if restartCount != pod.Status.ContainerStatuses[0].RestartCount {
			waitRestarted(pod, &restartCount)
			time.Sleep(1 * time.Second)
		}
		//reset connection if container restarts
		PodLogsConnection := podInterface.GetLogs(pod.Name, &v1.PodLogOptions{
			Follow:    true,
		})
		LogStream, err = PodLogsConnection.Stream(ctx)
		scanner = bufio.NewScanner(LogStream)
	}
}

func main(){
	config, err := rest.InClusterConfig()
	checkErr(err)

	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)

	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
    checkErr(err)

	api := clientset.CoreV1()
	podInterface := api.Pods(string(ns))
	ctx := context.Background()
	watcher, err := podInterface.Watch(ctx, metav1.ListOptions{})
    checkErr(err)

	r, _ := regexp.Compile(os.Getenv("NAME_FILTER"))

	for {
        select {
        case event := <-watcher.ResultChan():
        	pod, err := event.Object.(*v1.Pod)
        	if !err{log.Fatal("undefined")}
			switch event.Type {
				case watch.Added:
					if r.MatchString(pod.Name) { 
						fmt.Printf("Pod named %s added!\n", pod.Name) //optional
						wait.PollImmediateInfinite(time.Second, isPodRunning(podInterface, pod.Name, ctx))
						go getPodLogs(pod, podInterface, ctx)
					}
				case watch.Deleted:
					fmt.Printf("Pod named %s deleted!\n", pod.Name) //optional
				}
		}
	}
}
