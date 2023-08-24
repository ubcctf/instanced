package main

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func (in *Instancer) registerEndpoints() {
	in.echo.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "healthy")
	})

	in.echo.POST("/instances", func(c echo.Context) error {
		chalName := c.QueryParam("chal")
		token := c.QueryParam("token")

		manifest, ok := in.config.Challenges[chalName]
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
		// todo: create challenge
		var err error

		objYamls := strings.Split(manifest, "---")
		c.Logger().Infof("creating %d objects", len(objYamls))
		for _, v := range objYamls {
			if v == "\n" || v == "" {
				// ignore empty cases
				continue
			}
			resObj, err := in.CreateObject(v, "challenges")
			in.echo.Logger.Debug(resObj)
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
	})

	in.echo.DELETE("/instances/:id", func(c echo.Context) error {

		return c.JSON(http.StatusAccepted, "destroyed")
	})
}
