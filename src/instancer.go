package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

type Instancer struct {
	k8sConfig *rest.Config
	config    *Config
	echo      *echo.Echo
	// challengeObjs map[string][]unstructured.Unstructured
	challengeTmpls map[string]*template.Template
	db             *sql.DB
	log            zerolog.Logger
}

func InitInstancer() (*Instancer, error) {
	in := Instancer{
		echo: echo.New(),
	}
	in.echo.HideBanner = true
	in.echo.HidePort = true
	in.log = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.DebugLevel)
	in.echo.Logger = NewEchoLog(in.log)
	log := in.log.With().Str("component", "instanced-init").Logger()

	reqLog := log.With().Str("component", "echo-req").Logger()
	in.echo.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// Ignore healthcheck endpoint to prevent spam.
			if c.Path() == "healthz" {
				return nil
			}
			reqLog.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Msg("request")

			return nil
		},
	}))

	in.registerEndpoints()

	var err error
	in.config, err = loadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not load config")
	}
	log.Info().Int("count", len(in.config.Challenges)).Msg("read challenges from config")
	log.Debug().Str("value", in.config.ListenAddr).Msg("read listenAddr from config")

	/* 	in.challengeObjs, err = UnmarshalChallenges(in.config.Challenges)
	   	if err != nil {
	   		log.Fatal().Err(err).Msg("could not parse challenge manifests")
	   	}
	   	for k := range in.challengeObjs {
	   		log.Info().Str("challenge", k).Msg("parsed challenge manifest")
	   	}
	   	log.Info().Int("count", len(in.challengeObjs)).Msg("parsed challenges") */
	// Parse templates
	in.challengeTmpls = make(map[string]*template.Template, len(in.config.Challenges))
	for k, v := range in.config.Challenges {
		tmpl, err := template.New("challenge").Parse(v)
		if err != nil {
			log.Error().Err(err).Str("challenge", k).Msg("could not parse a challenge template")
			continue
		}
		in.challengeTmpls[k] = tmpl
	}

	in.k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not create kube-api client config")
	}
	rest.SetKubernetesDefaults(in.k8sConfig)
	log.Debug().Str("config", fmt.Sprintf("%+v", in.k8sConfig)).Msg("loaded kube-api client config")

	// Test CRDs
	log.Debug().Msg("querying CRDs")
	crdChallObjs, err := in.QueryInstancedChallenges("challenges")
	if err != nil {
		log.Debug().Err(err).Msg("error retrieving challenge definitions from CRDs")
	} else {
		for k, o := range crdChallObjs {
			log.Debug().Str("challenge", k).Msg("parsed challenge from CRD")
			for _, v := range o {
				log.Debug().Str("kind", v.GetKind()).Str("name", v.GetName()).Str("challenge", k).Msg("parsed resource")
			}
		}
	}

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
	log.Info().Int("count", len(instances)).Msg("instances found")
	for _, i := range instances {
		// Any does not marshall properly
		log.Debug().Int64("id", i.Id).Time("expiry", i.Expiry).Str("challenge", i.Challenge).Msg("instance record found")
		if time.Now().After(i.Expiry) {
			log.Info().Int64("id", i.Id).Str("challenge", i.Challenge).Msg("destroying expired instance")
			err := in.DestroyInstance(i)
			if err != nil {
				log.Error().Err(err).Msg("error destroying instance")
			}
		}
	}
}

func (in *Instancer) DestroyInstance(rec InstanceRecord) error {
	log := in.log.With().Str("component", "instanced").Logger()
	/* 	chal, ok := in.challengeObjs[rec.Challenge]
	   	if !ok {
	   		return &ChallengeNotFoundError{rec.Challenge}
	   	} */
	chal, err := in.GetChalObjsFromTemplate(rec.Challenge, rec.UUID)
	if err != nil {
		return err
	}

	for _, o := range chal {
		obj := o.DeepCopy()
		// todo: set proper name
		//obj.SetName(fmt.Sprintf("in-%v-%v", obj.GetName(), rec.Id))
		err := in.DeleteObject(obj, "challenges")
		if err != nil {
			log.Warn().Err(err).Str("name", obj.GetName()).Str("kind", obj.GetKind()).Msg("error deleting object")
		}
	}
	err = in.DeleteInstanceRecord(rec.Id)
	if err != nil {
		log.Warn().Err(err).Msg("error deleting instance record")
	}
	return nil
}

func (in *Instancer) CreateInstance(challenge, team string) (InstanceRecord, error) {
	log := in.log.With().Str("component", "instanced").Logger()

	/* chal, ok := in.challengeObjs[challenge]
	if !ok {
		return InstanceRecord{}, &ChallengeNotFoundError{challenge}
	} */
	cuuid := uuid.NewString()[0:8]
	chal, err := in.GetChalObjsFromTemplate(challenge, cuuid)
	if err != nil {
		return InstanceRecord{}, err
	}

	ttl, err := time.ParseDuration(in.config.InstanceTTL)
	if err != nil {
		log.Warn().Err(err).Msg("could not parse instance ttl, defaulting to 10 minutes")
		ttl = 10 * time.Minute
	}

	rec, err := in.InsertInstanceRecord(ttl, team, challenge, cuuid)
	if err != nil {
		log.Error().Err(err).Msg("could not create instance record")
	} else {
		log.Info().Time("expiry", rec.Expiry).
			Str("challenge", rec.Challenge).
			Int64("id", rec.Id).
			Msg("registered new instance")
	}

	var createErr error
	log.Info().Int("count", len(chal)).Msg("creating objects")
	for _, o := range chal {
		obj := o.DeepCopy()
		//obj.SetName(fmt.Sprintf("in-%v-%v", obj.GetName(), rec.Id))
		var resObj *unstructured.Unstructured
		resObj, createErr = in.CreateObject(obj, "challenges")
		log.Debug().Any("object", resObj).Msg("created object")
		if createErr != nil {
			log.Error().Err(createErr).Msg("error creating object")
			break
		}
		log.Info().Str("kind", resObj.GetKind()).Str("name", resObj.GetName()).Msg("created object")
	}
	if createErr != nil {
		// todo: handle errors/cleanup incomplete deploys?
		log.Error().Err(err).Msg("could not create an object")
		log.Info().Msg("instance creation incomplete, manual intervention required")
		return InstanceRecord{}, errors.New("instance deployment failed")
	}
	return rec, nil
}

func (in *Instancer) GetTeamChallengeStates(teamID string) ([]InstanceRecord, error) {
	instances, err := in.ReadInstanceRecordsTeam(teamID)
	if err != nil {
		return nil, err
	}
	//for k := range in.challengeObjs {
	for k := range in.config.Challenges {
		active := false
		for _, v := range instances {
			if v.Challenge == k {
				active = true
				break
			}
		}
		if !active {
			instances = append(instances, InstanceRecord{Expiry: time.Unix(0, 0), Challenge: k, TeamID: teamID})
		}
	}
	return instances, nil
}

type ChalInstIdentifier struct {
	ID string
}

func (in *Instancer) GetChalObjsFromTemplate(chalName string, cuuid string) ([]unstructured.Unstructured, error) {
	tmpl, ok := in.challengeTmpls[chalName]
	if !ok {
		return nil, &ChallengeNotFoundError{chalName}
	}
	var objstr bytes.Buffer
	tmpl.Execute(&objstr, ChalInstIdentifier{ID: cuuid})
	chal, err := UnmarshalManifestFile(objstr.String())
	if err != nil {
		return nil, fmt.Errorf("could not parse challenge: %q : %w", chalName, err)
	}
	return chal, nil
}

func (in *Instancer) ParseTemplates() {
	in.challengeTmpls = make(map[string]*template.Template, len(in.config.Challenges))
	for k, v := range in.config.Challenges {
		tmpl, err := template.New("challenge").Parse(v)
		if err != nil {
			in.log.Error().Err(err).Str("challenge", k).Msg("could not parse a challenge template")
			continue
		}
		in.challengeTmpls[k] = tmpl
	}
}

func (in *Instancer) Start() error {
	log := in.log.With().Str("component", "instanced").Logger()
	log.Info().Msg("starting webserver...")
	// Start Webserver
	go in.echo.Start(in.config.ListenAddr)

	// Ticker to read db for expired instances
	log.Info().Msg("starting instance monitoring loop")
	// todo: some sort of garbage collection for instances that we somehow lost track of
	for {
		log.Info().Msg("checking for expired instances...")
		in.DestoryExpiredInstances()
		time.Sleep(time.Second * 60)
	}
}
