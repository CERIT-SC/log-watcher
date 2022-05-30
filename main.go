/*========================
		Log-watcher
	author: Patrik Jesko
	last update: 30/05/22
==========================*/

/*
Need fixing:
pod not writing ouput longer than 30s -> err //problem on kubernetes side I think
when container restarts (depends how much it takes to restart) -> reading the old log till the new conainer isnt ready
not tested with lots of pods
*/


package main

import (
	"fmt"
	"context"
	"time"
	"log"
	"bufio"
	"os"
	//"regexp"
	"sync"
	"strings"
	"path/filepath"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func waitRestarted(pod *v1.Pod, restartCount *int32){
	for {
		if pod.Status.ContainerStatuses[0].RestartCount > *restartCount {
			*restartCount = pod.Status.ContainerStatuses[0].RestartCount
			log.Println(*restartCount)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

//Getting pod logs
func getPodLogs(pod *v1.Pod, podInterface kv1.PodInterface, ctx context.Context, wg *sync.WaitGroup){
	fmt.Printf("Now watching pod %s\n", pod.Name) //just testing
	defer wg.Done()

	//file, err := os.Create("/tmp/"+pod.Name+".txt")
	//checkErr(err)
	//defer file.Close()
	log.Printf("file named "+pod.Name+".txt created!") //testing
	PodLogsConnection := podInterface.GetLogs(pod.Name, &v1.PodLogOptions{
		Follow:    true,
	})
	LogStream, err := PodLogsConnection.Stream(ctx)
	checkErr(err)
	defer LogStream.Close()
	defer fmt.Printf("Stoped watching %s\n", pod.Name) //very important to see //testing
	scanner := bufio.NewScanner(LogStream)
	var line string	
	var restartCount int32 = pod.Status.ContainerStatuses[0].RestartCount

	for {//this needs a lot of improvement
		for scanner.Scan() { //returns bool so if Scan is false (when scan stops)
			line = scanner.Text()
			if strings.Contains(line, "unable to retrieve container logs for containerd:") || strings.Contains(line, "an error occurred when try to find container") { 
				//true if .Text() returns token with errors in condition
				return
			} //temporary solution but gets the job done
			//_, err = file.WriteString(line)
			fmt.Printf(line)
			//checkErr(err)
		}
		//out of scan
		if GetPodStatus(podInterface, pod.Name, ctx) || scanner.Err() != nil{ 
			//true if deleted or finished(job) || scaner has error
			return
		}
		if restartCount != pod.Status.ContainerStatuses[0].RestartCount {
			//wait a while till the container fully restarts (still not enough time)
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
	/*
	//creating in-cluster config
	config, err := rest.InClusterConfig()
	checkErr(err)
	//creating clientset
	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)
	*/
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
    )
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    checkErr(err)
	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)
	
	ns := "jesko-ns"
	//ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace") //read the namespace name
    checkErr(err)
	api := clientset.CoreV1()
	podInterface := api.Pods(string(ns))
	checkErr(err)
	ctx := context.Background()
	watcher, err := podInterface.Watch(ctx, metav1.ListOptions{})
    checkErr(err)
    //ch := watcher.ResultChan()
	var wg sync.WaitGroup

	//r, _ := regexp.Compile(os.Getenv("FILTER"))

	for {
        select {
        case event := <-watcher.ResultChan():
        	pod, err := event.Object.(*v1.Pod)
        	if !err{log.Fatal("undefined")} //fatal is risky..
			switch event.Type {
				case watch.Added:
					if strings.Contains(pod.Name, "hello") {//r.MatchString(pod.Name) {
						fmt.Printf("Pod named %s added!\n", pod.Name) //optional
						wait.PollImmediateInfinite(time.Second, isPodRunning(podInterface, pod.Name, ctx))
						wg.Add(1)
						go getPodLogs(pod, podInterface, ctx, &wg)
					}
				case watch.Deleted:
					fmt.Printf("Pod named %s deleted!\n", pod.Name) //optional
				}
		}
		//time.Sleep(1 * time.Milliseconds)
	}
	wg.Wait()
}