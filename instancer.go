package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

type Instancer struct {
	k8sConfig     *rest.Config
	config        *Config
	echo          *echo.Echo
	challengeObjs map[string][]unstructured.Unstructured
	db            *sql.DB
	log           zerolog.Logger
}

func InitInstancer() (*Instancer, error) {
	in := Instancer{
		echo: echo.New(),
	}
	in.echo.HideBanner = true
	in.log = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.DebugLevel)
	in.echo.Logger = NewEchoLog(in.log)
	log := in.log.With().Str("component", "instanced-init").Logger()

	in.registerEndpoints()

	var err error
	in.config, err = loadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not load config")
	}
	log.Info().Int("count", len(in.config.Challenges)).Msg("read challenges from config")
	log.Debug().Str("value", in.config.ListenAddr).Msg("read listenAddr from config")

	in.challengeObjs, err = UnmarshalChallenges(in.config.Challenges)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse challenge manifests")
	}
	for k := range in.challengeObjs {
		log.Info().Str("challenge", k).Msg("parsed challenge manifest")
	}
	log.Info().Int("count", len(in.challengeObjs)).Msg("parsed challenges")

	in.k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not create kube-api client config")
	}
	rest.SetKubernetesDefaults(in.k8sConfig)
	log.Debug().Str("config", fmt.Sprintf("%+v", in.k8sConfig)).Msg("loaded kube-api client config")

	err = in.InitDB("/data/instancer.db")
	if err != nil {
		log.Fatal().Err(err).Msg("could not init sqlite db")
	}
	log.Info().Msg("initialized database")

	return &in, nil
}

func (in *Instancer) DestoryExpiredInstances() {
	log := in.log.With().Str("component", "instanced").Logger()
	instances, err := in.ReadInstanceRecords()
	if err != nil {
		log.Error().Err(err).Msg("error reading instance records")
		return
	}
	log.Info().Int("count", len(instances)).Msg("found instances")
	for _, i := range instances {
		log.Debug().Any("record", i).Msg("found instance record")
		if time.Now().After(i.expiry) {
			log.Info().Int("id", i.id).Msg("destroying expired instance")
			err := in.DestroyInstance(i)
			if err != nil {
				log.Error().Err(err).Msg("error destroying instance")
			}
		}
	}
}

func (in *Instancer) DestroyInstance(rec InstanceRecord) error {
	log := in.log.With().Str("component", "instanced").Logger()
	chal, ok := in.challengeObjs[rec.challenge]
	if !ok {
		return fmt.Errorf("manifest for challenge %v not in memory", rec.challenge)
	}
	for _, o := range chal {
		obj := o.DeepCopy()
		// todo: set proper name
		obj.SetName("instancer-test")
		err := in.DeleteObject(obj, "challenges")
		if err != nil {
			log.Warn().Err(err).Str("name", obj.GetName()).Str("kind", obj.GetKind()).Msg("error deleting object")
		}
	}
	err := in.DeleteInstanceRecord(rec.id)
	if err != nil {
		log.Warn().Err(err).Msg("error deleting instance record")
	}
	return nil
}

func (in *Instancer) Start() error {
	log := in.log.With().Str("component", "instanced").Logger()
	log.Info().Msg("starting webserver...")
	// Start Webserver
	go in.echo.Start(":8080")

	// Ticker to read db for expired instances
	log.Info().Msg("starting instance monitoring loop")
	// todo: some sort of garbage collection for instances that we somehow lost track of
	for {
		log.Info().Msg("checking for expired instances...")
		in.DestoryExpiredInstances()
		time.Sleep(time.Second * 60)
	}
}
