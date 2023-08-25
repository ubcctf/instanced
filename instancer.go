package main

import (
	"database/sql"
	"time"

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
	db            *sql.DB
}

func InitInstancer() (*Instancer, error) {
	in := Instancer{
		echo: echo.New(),
	}
	in.echo.Logger.SetLevel(log.DEBUG)
	in.registerEndpoints()

	var err error
	in.config, err = loadConfig()
	if err != nil {
		in.echo.Logger.Fatalf("error loading config: %s", err)
	}

	in.challengeObjs, err = in.UnmarshalChallenges(in.config.Challenges)
	if err != nil {
		in.echo.Logger.Fatalf("error unmarshalling challenges: %s", err)
	}

	in.k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		in.echo.Logger.Fatalf("error creating k8s configuration: %s", err)
	}
	rest.SetKubernetesDefaults(in.k8sConfig)

	in.k8sClientSet, err = kubernetes.NewForConfig(in.k8sConfig)
	if err != nil {
		in.echo.Logger.Fatalf("error initializing client: %s", err)
	}

	return &in, nil
}

func (in *Instancer) DestoryExpiredInstances() {

}

func (in *Instancer) DestroyInstance(id int) error {
	return nil
}

func (in *Instancer) Start() error {
	// Start Webserver
	go in.echo.Logger.Fatal(in.echo.Start(in.config.ListenAddr))

	// Ticker to read db for expired instances
	checkExpiry := time.NewTicker(time.Minute).C

	// todo: some sort of garbage collection for instances that we somehow lost track of
	for {
		<-checkExpiry
		in.DestoryExpiredInstances()
	}
}
