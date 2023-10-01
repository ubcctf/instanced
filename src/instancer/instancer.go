package instancer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/ubcctf/instanced/src/db"
	"github.com/ubcctf/instanced/src/k8s"
)

type Instancer struct {
	k8sC k8s.KubeClient
	dbC  db.DBClient
	srv  *echo.Echo
	// challengeObjs map[string][]unstructured.Unstructured
	challengeTmpls map[string]*template.Template
	conf           Config
	log            zerolog.Logger
}

func InitInstancer() *Instancer {
	in := Instancer{}

	// Initial Logger
	in.log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.TraceLevel)

	// Load Config
	in.conf = loadConfig(in.log)

	// Set Config Log Level
	in.log = in.log.Level(in.conf.LogLevel)

	// Set and configure API server
	in.srv = initWebServer(in.log, in.conf.LogRequests)
	in.registerRequestHandlers()

	log := in.log.With().Str("component", "instanced-init").Logger()

	// Load In-Cluster kube client config
	var err error
	in.k8sC, err = k8s.NewKubeClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed loading kube-client in-cluster config")
	}
	log.Debug().Str("config", fmt.Sprintf("%+v", in.k8sC)).Msg("loaded kube-api client config")

	// Open DB connection
	in.dbC, err = db.InitDB(in.conf.DBFile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed opening sqlite database")
	}

	return &in
}

func (in *Instancer) Start() {
	log := in.log.With().Str("component", "instanced").Logger()
	log.Info().Msg("starting webserver...")
	// Start Webserver
	go func() {
		if err := in.srv.Start(in.conf.ListenAddr); err != nil {
			log.Fatal().Err(err).Msg("failed to start api server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	log.Info().Msg("starting instance monitoring loop")

	// Ticker to read db for expired instances
	checkExpired := time.NewTicker(time.Second * 60)
	defer checkExpired.Stop()

	for {
		select {
		case <-checkExpired.C:
			// todo, add cancel to this
			log.Info().Msg("checking for expired instances...")
			go in.DestoryExpiredInstances()

		case <-quit:
			// Graceful shutdown http server
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := in.srv.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("failed graceful shutdown")
			}
			return
		}
	}
}
