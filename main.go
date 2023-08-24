package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
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
	config       *Config
}

type Config struct {
	// map[name]manifest in yaml. supports multiple objects per file delimited with ---
	Challenges map[string]string `json:"challenges"`
	ListenAddr string            `json:"listenAddr"`
}

func main() {
	instancer := Instancer{
		echo: echo.New(),
	}
	instancer.registerEndpoints()
	var err error
	instancer.config, err = loadConfig()
	if err != nil {
		instancer.echo.Logger.Fatalf("error: %s", err)
	}
	instancer.k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		instancer.echo.Logger.Fatalf("error creating k8s configuration: %s", err)
	}
	rest.SetKubernetesDefaults(instancer.k8sConfig)
	instancer.k8sClientSet, err = kubernetes.NewForConfig(instancer.k8sConfig)
	if err != nil {
		instancer.echo.Logger.Fatalf("error initializing client: %s", err)
	}

	go instancer.echo.Logger.Fatal(instancer.echo.Start(":8080"))

	select {}
}

func loadConfig() (*Config, error) {
	confb, err := os.ReadFile("/config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("when reading config file:\n\t%s", err)
	}
	conf := &Config{}
	err = yaml.Unmarshal(confb, conf)
	if err != nil {
		return nil, fmt.Errorf("when parsing config file:\n\t%s", err)
	}
	return conf, nil
}

func (in *Instancer) registerEndpoints() {
	in.echo.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "healthy")
	})

	in.echo.POST("/instances", func(c echo.Context) error {
		chalName := c.QueryParam("chal")
		token := c.QueryParam("token")

		manifest, ok := in.config.Challenges[chalName]
		if !ok {
			// todo: don't sprintf user controlled data
			c.Logger().Infof("request rejected with invalid challenge %s", chalName)
			return c.JSON(http.StatusNotFound, "challenge not supported")
		}
		// todo: check an auth token or something
		if token == "" {
			c.Logger().Info("request rejected with no token")
			return c.JSON(http.StatusForbidden, "team token not provided")
		}
		// todo: create challenge
		var err error
		objs := parseK8sYaml(manifest)
		for _, obj := range objs {
			_, err = createObject(
				in.k8sClientSet,
				in.k8sConfig,
				obj)
			if err != nil {
				break
			}
		}
		if err != nil {
			// todo: handle errors/cleanup incomplete deploys?
			c.Logger().Errorf("could create an object: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, "challenge deploy failed: contact admin")
		}
		c.Logger().Info("provisioned new instance")
		return c.JSON(http.StatusAccepted, "created")
	})

	in.echo.DELETE("/instances/:id", func(c echo.Context) error {
		/*_, err = clientset.CoreV1().Pods("default").Get(context.TODO(), "example-xxxxx", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Printf("Pod example-xxxxx not found in default namespace\n")
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, "deletion failed: contact admin")
		} else {
			fmt.Printf("Found example-xxxxx pod in default namespace\n")
		}*/
		return c.JSON(http.StatusAccepted, "destroyed")
	})
}

// https://github.com/kubernetes/client-go/issues/193#issuecomment-363318588
func parseK8sYaml(manifest string) []runtime.Object {
	sepYamlfiles := strings.Split(manifest, "---")
	result := make([]runtime.Object, 0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(f), nil, nil)
		if err != nil {
			log.Printf("Error while decoding YAML object. Err was: %s", err)
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
	restClient, err := newRestClient(config, mapping.GroupVersionKind.GroupVersion())
	if err != nil {
		return nil, err
	}

	// Use the REST helper to create the object in the "default" namespace.
	restHelper := resource.NewHelper(restClient, mapping)
	return restHelper.Create("challenges", false, obj)
}

func newRestClient(restConfig *rest.Config, gv schema.GroupVersion) (rest.Interface, error) {
	restConfig.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	restConfig.GroupVersion = &gv
	if len(gv.Group) == 0 {
		restConfig.APIPath = "/api"
	} else {
		restConfig.APIPath = "/apis"
	}

	return rest.RESTClientFor(restConfig)
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
		/*


		time.Sleep(20 * time.Second)
	}
}
*/
