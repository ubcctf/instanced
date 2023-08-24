package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Instancer struct {
	k8sConfig     *rest.Config
	k8sClientSet  *kubernetes.Clientset
	config        *Config
	echo          *echo.Echo
	challengeObjs map[string][]unstructured.Unstructured
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
		instancer.echo.Logger.Fatalf("error loading config: %s", err)
	}

	instancer.challengeObjs, err = instancer.UnmarshalChallenges(instancer.config.Challenges)
	if err != nil {
		instancer.echo.Logger.Fatalf("error unmarshalling challenges: %s", err)
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

	go instancer.echo.Logger.Fatal(instancer.echo.Start(instancer.config.ListenAddr))

	select {}
}
