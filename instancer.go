package main

import (
	"database/sql"
	"fmt"
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
	instances, err := in.ReadInstanceRecords()
	if err != nil {
		in.echo.Logger.Errorf("error reading instance records: %s", err)
	}
	for _, i := range instances {
		if time.Now().After(i.expiry) {
			in.echo.Logger.Infof("destroying instance %s", i.id)
			err := in.DestroyInstance(i)
			if err != nil {
				in.echo.Logger.Errorf("error destroying instance: %s", err)
			}
		}
	}
}

func (in *Instancer) DestroyInstance(rec InstanceRecord) error {
	chal, ok := in.challengeObjs[rec.challenge]
	if !ok {
		return fmt.Errorf("manifest for challenge %s not in memory", rec.challenge)
	}
	for _, o := range chal {
		obj := o.DeepCopy()
		// todo: set proper name
		obj.SetName("instancer-test")
		err := in.DeleteObject(obj, "challenges")
		if err != nil {
			in.echo.Logger.Warnf("error deleting object %s: %s", obj.GetName(), err)
		}
	}
	err := in.DeleteInstanceRecord(rec.id)
	if err != nil {
		in.echo.Logger.Warnf("error deleting instance record: %s", err)
	}
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
		in.echo.Logger.Info("checking for expired instances...")
		in.DestoryExpiredInstances()
	}
}
