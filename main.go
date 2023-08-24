package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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
	instancer.echo.Logger.SetLevel(log.DEBUG)
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

func (in *Instancer) CreateObject(manifest string) (*unstructured.Unstructured, error) {
	unstructObj, err := ParseK8sYaml(manifest)
	if err != nil {
		return nil, err
	}
	in.echo.Logger.Debug(unstructObj)

	c, err := discovery.NewDiscoveryClientForConfig(in.k8sConfig)
	if err != nil {
		return nil, err
	}

	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := mapper.RESTMapping(unstructObj.GetObjectKind().GroupVersionKind().GroupKind())
	if err != nil {
		return nil, err
	}
	client, err := dynamic.NewForConfig(in.k8sConfig)
	if err != nil {
		return nil, err
	}
	resObj, err := client.Resource(mapping.Resource).Namespace("challenges").Create(context.Background(), unstructObj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return resObj, nil
}

func SplitYaml(content string) []string {
	return strings.Split(content, "---")
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

		objYamls := strings.Split(manifest, "---")
		c.Logger().Infof("creating %d objects", len(objYamls))
		for _, v := range objYamls {
			if v == "\n" || v == "" {
				// ignore empty cases
				continue
			}
			resObj, err := in.CreateObject(v)
			in.echo.Logger.Debug(resObj)
			if err != nil {
				break
			}
			c.Logger().Infof("created %s named: %s", resObj.GetKind(), resObj.GetName())
		}
		if err != nil {
			// todo: handle errors/cleanup incomplete deploys?
			c.Logger().Errorf("could not create an object: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, "challenge deploy failed: contact admin")
		}
		c.Logger().Info("provisioned new instance")
		return c.JSON(http.StatusAccepted, "created")
	})

	in.echo.DELETE("/instances/:id", func(c echo.Context) error {

		return c.JSON(http.StatusAccepted, "destroyed")
	})
}

// https://github.com/kubernetes/client-go/issues/193#issuecomment-363318588
func ParseK8sYaml(manifest string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	err := yaml.UnmarshalStrict([]byte(manifest), obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
