package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// Register web request handlers
func (in *Instancer) registerEndpoints() {
	in.echo.GET("/healthz", in.handleLivenessCheck)

	in.echo.GET("/instances", in.handleInstanceList)

	in.echo.POST("/instances", in.handleInstanceCreate)

	in.echo.DELETE("/instances", in.handleInstanceDelete)
}

type InstancesResponse struct {
	Action    string `json:"action"`
	Challenge string `json:"challenge"`
	ID        int64  `json:"id"`
}

func (in *Instancer) handleLivenessCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, "healthy")
}

func (in *Instancer) handleInstanceCreate(c echo.Context) error {
	chalName := c.QueryParam("chal")
	token := c.QueryParam("token")

	// todo: check an auth token or something
	if token == "" {
		c.Logger().Info("request rejected with no token")
		return c.JSON(http.StatusForbidden, "team token not provided")
	}
	rec, err := in.CreateInstance(chalName)
	if _, ok := err.(*ChallengeNotFoundError); ok {
		return c.JSON(http.StatusNotFound, "challenge not supported")
	}

	if err != nil {
		// todo: handle errors/cleanup incomplete deploys?
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, "challenge deploy failed: contact admin")
	}
	c.Logger().Info("processed request to provision new instance")
	return c.JSON(http.StatusAccepted, InstancesResponse{"created", chalName, rec.id})
}

func (in *Instancer) handleInstanceDelete(c echo.Context) error {
	chalName := c.QueryParam("chal")
	token := c.QueryParam("token")
	instanceID, err := strconv.ParseInt(c.QueryParam("id"), 10, 64)

	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	// todo: check an auth token or something
	if token == "" {
		c.Logger().Info("request rejected with no token")
		return c.JSON(http.StatusForbidden, "team token not provided")
	}

	err = in.DestroyInstance(InstanceRecord{instanceID, time.Now(), chalName})

	if _, ok := err.(*ChallengeNotFoundError); ok {
		return c.JSON(http.StatusNotFound, "challenge not supported")
	}

	if err != nil {
		// todo: handle errors/cleanup incomplete deploys?
		c.Logger().Errorf("request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, "challenge destroy failed: contact admin")
	}
	c.Logger().Info("processed request to destroy an instance")

	return c.JSON(http.StatusAccepted, InstancesResponse{"destroyed", chalName, instanceID})
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
