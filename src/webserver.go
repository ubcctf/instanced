package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Register web request handlers
func (in *Instancer) registerEndpoints() {
	in.echo.GET("/healthz", in.handleLivenessCheck)

	in.echo.GET("/instances", in.handleInstanceList)

	in.echo.POST("/instances", in.handleInstanceCreate)

	in.echo.DELETE("/instances", in.handleInstanceDelete)

	in.echo.GET("/challenges", in.handleInstanceListTeam)
}

type InstancesResponse struct {
	Action    string `json:"action"`
	Challenge string `json:"challenge"`
	ID        int64  `json:"id"`
	URL       string `json:"url"`
}

func (in *Instancer) handleLivenessCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, "healthy")
}

func (in *Instancer) handleInstanceCreate(c echo.Context) error {
	chalName := c.QueryParam("chal")
	teamID := c.QueryParam("team")

	recs, err := in.ReadInstanceRecordsTeam(teamID)
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
	return c.JSON(http.StatusAccepted, InstancesResponse{"created", chalName, rec.Id, fmt.Sprintf("http://%v.%v.ctf.maplebacon.org", rec.UUID, chalName)})
}

func (in *Instancer) handleInstanceDelete(c echo.Context) error {
	if !c.QueryParams().Has("id") {
		return in.handleInstancePurge(c)
	}
	instanceID, err := strconv.ParseInt(c.QueryParam("id"), 10, 64)

	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	rec, err := in.ReadInstanceRecord(instanceID)

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
	recs, err := in.ReadInstanceRecords()
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
	records, err := in.ReadInstanceRecords()
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
