package main

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("error creating configuration: %s", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("error initializing client: %s", err)
	}

	for {
		// get pods in all the namespaces by omitting namespace
		// Or specify namespace to get pods in particular namespace
		pods, err := clientset.CoreV1().Pods("challenges").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		// Examples for error handling:
		// - Use helper functions e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		/* 		_, err = clientset.CoreV1().Pods("default").Get(context.TODO(), "example-xxxxx", metav1.GetOptions{})
		   		if errors.IsNotFound(err) {
		   			fmt.Printf("Pod example-xxxxx not found in default namespace\n")
		   		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		   			fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
		   		} else if err != nil {
		   			panic(err.Error())
		   		} else {
		   			fmt.Printf("Found example-xxxxx pod in default namespace\n")
		   		} */

		time.Sleep(20 * time.Second)
	}
}
