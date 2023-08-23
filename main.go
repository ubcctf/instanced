package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type Instancer struct {
	echo         *echo.Echo
	k8sConfig    *rest.Config
	k8sClientSet *kubernetes.Clientset
}

func main() {
	instancer := Instancer{
		echo: echo.New(),
	}
	registerEndpoints(instancer.echo)
	var err error
	instancer.k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		instancer.echo.Logger.Fatalf("error creating k8s configuration: %s", err)
	}
	instancer.k8sClientSet, err = kubernetes.NewForConfig(instancer.k8sConfig)
	if err != nil {
		log.Fatalf("error initializing client: %s", err)
	}

	go instancer.echo.Logger.Fatal(instancer.echo.Start(":8080"))

	select {}
}

func registerEndpoints(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Healthy")
	})

	e.POST("/instances", func(c echo.Context) error {
		// Get challenge name
		// Check config for challenge

		return c.String(http.StatusAccepted, "Creating")
	})

	e.POST("/instances/[id]/destroy", func(c echo.Context) error {
		return c.String(http.StatusAccepted, "Destroying")
	})
}

// https://github.com/kubernetes/client-go/issues/193#issuecomment-363318588
func parseK8sYaml(fileR []byte) []runtime.Object {
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	result := make([]runtime.Object, 0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(f), nil, nil)
		if err != nil {
			log.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
			continue
		}
		result = append(result, obj)
	}
	return result
}

func createObject(clientSet kubernetes.Interface, config *rest.Config, obj runtime.Object) (runtime.Object, error) {
	// Create a REST mapper that tracks information about the available resources in the cluster.
	groupResources, err := restmapper.GetAPIGroupResources(clientSet.Discovery())
	if err != nil {
		return nil, err
	}
	rm := restmapper.NewDiscoveryRESTMapper(groupResources)

	// Get some metadata needed to make the REST request.
	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := rm.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}
	// Create a client specifically for creating the object.
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	// Use the REST helper to create the object in the "default" namespace.
	restHelper := resource.NewHelper(restClient, mapping)
	return restHelper.Create("default", false, obj)
}

/*
func watchPods(log echo.Logger) {
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
		log.Printf("There are %d pods in the cluster\n", len(pods.Items))
		for _, p := range pods.Items {
			log.Print(p.ObjectMeta.Name)
		}

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
		   		}


		time.Sleep(20 * time.Second)
	}
}
*/
