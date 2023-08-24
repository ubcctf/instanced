package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Register web request handlers
func (in *Instancer) registerEndpoints() {
	in.echo.GET("/healthz", in.handleLivenessCheck)

	in.echo.POST("/instances", in.handleInstanceCreate)

	in.echo.DELETE("/instances", in.handleInstanceDelete)
}

func (in *Instancer) handleLivenessCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, "healthy")
}

func (in *Instancer) handleInstanceCreate(c echo.Context) error {
	chalName := c.QueryParam("chal")
	token := c.QueryParam("token")

	templateObjs, ok := in.challengeObjs[chalName]
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
	var err error

	// todo: copy and rename template objects
	c.Logger().Infof("creating %d objects", len(templateObjs))
	for _, o := range templateObjs {
		resObj, err := in.CreateObject(&o, "challenges")
		in.echo.Logger.Debugf("creating %s", resObj)
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
}

func (in *Instancer) handleInstanceDelete(c echo.Context) error {
	//chalName := c.QueryParam("chal")
	//instanceID := c.QueryParam("id")
	//token := c.QueryParam("token")

	return c.JSON(http.StatusAccepted, "destroyed")
}
