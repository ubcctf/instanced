package instancer

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/ubcctf/instanced/src/adapters"
)

type InstancesResponse struct {
	Action    string `json:"action"`
	Challenge string `json:"challenge"`
	ID        int64  `json:"id"`
	URL       string `json:"url"`
}

func initWebServer(log zerolog.Logger, logRequests bool) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger = adapters.NewEchoLog(log)

	// Register request logging middleware
	if logRequests {
		reqLog := log.With().Str("component", "echo-req").Logger()
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogURI:    true,
			LogStatus: true,
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				// Ignore healthcheck endpoint to prevent spam.
				if c.Path() == "/healthz" {
					return nil
				}
				reqLog.Info().
					Str("URI", v.URI).
					Int("status", v.Status).
					Msg("request")
				return nil
			},
		}))
	}

	e.Use(echoprometheus.NewMiddleware("instanced"))
	e.GET("/metrics", echoprometheus.NewHandler())

	return e
}

func (in *Instancer) registerRequestHandlers() {
	// Register requst handlers
	in.srv.GET("/healthz", in.handleLivenessCheck)
	in.srv.GET("/instances", in.handleInstanceList)
	in.srv.POST("/instances", in.handleInstanceCreate)
	in.srv.DELETE("/instances", in.handleInstanceDelete)
	in.srv.GET("/challenges", in.handleInstanceListTeam)
	in.srv.POST("/reload", in.handleCRDReload)
}

func (in *Instancer) handleLivenessCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, "healthy")
}

func (in *Instancer) handleInstanceCreate(c echo.Context) error {
	chalName := c.QueryParam("chal")
	teamID := c.QueryParam("team")

	recs, err := in.dbC.ReadInstanceRecordsTeam(teamID)
	if err != nil {
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, "challenge deploy failed: contact admin")
	}
	for _, r := range recs {
		if r.Challenge == chalName {
			return c.JSON(http.StatusTooManyRequests, "instance already exists for this challenge")
		}
	}

	rec, err := in.CreateInstance(chalName, teamID)
	if _, ok := err.(*ChallengeNotFoundError); ok {
		return c.JSON(http.StatusNotFound, "challenge not supported")
	}

	if err != nil {
		// todo: handle errors/cleanup incomplete deploys?
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, "challenge deploy failed: contact admin")
	}
	c.Logger().Info("processed request to provision new instance")
	return c.JSON(http.StatusAccepted, InstancesResponse{"created", chalName, rec.Id, fmt.Sprintf("https://%v.%v.ctf.maplebacon.org", rec.UUID, chalName)})
}

func (in *Instancer) handleInstanceDelete(c echo.Context) error {
	if !c.QueryParams().Has("id") {
		return in.handleInstancePurge(c)
	}
	instanceID, err := strconv.ParseInt(c.QueryParam("id"), 10, 64)

	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	rec, err := in.dbC.ReadInstanceRecord(instanceID)

	if err != nil {
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusNotFound, "instance id not found")
	}

	err = in.DestroyInstance(rec)

	if _, ok := err.(*ChallengeNotFoundError); ok {
		return c.JSON(http.StatusNotFound, "challenge not supported")
	}

	if err != nil {
		// todo: handle errors/cleanup incomplete deploys?
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, "challenge destroy failed: contact admin")
	}
	c.Logger().Info("processed request to destroy an instance")

	return c.JSON(http.StatusAccepted, InstancesResponse{"destroyed", rec.Challenge, instanceID, "TODO"})
}

func (in *Instancer) handleInstancePurge(c echo.Context) error {
	recs, err := in.dbC.ReadInstanceRecords()
	if err != nil {
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusNotFound, "instance id not found")
	}
	go func() {
		for _, r := range recs {
			err = in.DestroyInstance(r)
			if err != nil {
				c.Logger().Error("an instance failed to purge")
			}
		}
	}()

	return c.JSON(http.StatusAccepted, "instance purge started")
}

func (in *Instancer) handleInstanceList(c echo.Context) error {
	// todo: authenticate
	records, err := in.dbC.ReadInstanceRecords()
	if err != nil {
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, "request failed")
	}
	// todo: properly marshal records
	return c.JSON(http.StatusOK, records)
}

func (in *Instancer) handleInstanceListTeam(c echo.Context) error {
	teamID := c.QueryParam("team")
	records, err := in.GetTeamChallengeStates(teamID)
	if err != nil {
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, "request failed")
	}
	// todo: properly marshal records
	return c.JSON(http.StatusOK, records)
}

func (in *Instancer) handleCRDReload(c echo.Context) error {
	go in.LoadCRDs(context.TODO())
	return c.JSON(http.StatusAccepted, "accepted")
}
