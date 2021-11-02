package api

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/oid-explorer/api.oid-explorer.com/database"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type apiError struct {
	Error string `json:"error"`
}

func StartAPI() {
	log.Debug().Msg("starting the server")

	_, err := database.GetDB()
	if err != nil {
		log.Fatal().Err(err).Msg("connecting to the database failed")
	}

	e := echo.New()
	e.GET("/oids", searchOID)
	e.GET("/oids/:oid", getOID)
	e.GET("/oids/:oid/relation", getOIDRelation)
	e.GET("/oids/:oid/parent", getOIDParent)
	e.GET("/oids/:oid/siblings", getOIDSiblings)
	e.GET("/oids/:oid/children", getOIDChildren)

	// Start server
	go func() {
		err := e.Start(":" + viper.GetString("port"))

		if err != nil && err == http.ErrServerClosed {
			log.Info().Msg("shutting down the server")
		} else {
			log.Fatal().Err(err).Msg("unexpected server error")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Debug().Msg("received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = e.Shutdown(ctx); err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("shutting down the server failed")
	}
}

func searchOID(ctx echo.Context) error {
	var oidSearch database.OIDSearch

	keyword := ctx.QueryParam("keyword")
	if keyword != "" {
		queryType := ctx.QueryParam("type")
		switch queryType {
		case "oid":
			oidSearch.OID = &keyword
		case "name":
			oidSearch.Name = &keyword
		case "any", "":
			oidSearch.Any = &keyword
		default:
			return ctx.JSON(http.StatusBadRequest, apiError{"invalid queryType"})
		}
	}

	limit := ctx.QueryParam("limit")
	if limit != "" {
		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, apiError{errors.Wrap(err, "limit is not an integer").Error()})
		}
		oidSearch.Limit = &limitInt
	}

	db, err := database.GetDB()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, apiError{errors.Wrap(err, "failed to get db").Error()})
	}
	oids, err := db.SearchOID(oidSearch)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, apiError{err.Error()})
	}
	if len(oids) == 0 {
		return ctx.JSON(http.StatusNotFound, apiError{"no result"})
	}
	return ctx.JSON(http.StatusOK, oids)
}

func getOID(ctx echo.Context) error {
	wantedOID := ctx.Param("oid")
	db, err := database.GetDB()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, apiError{errors.Wrap(err, "failed to get db").Error()})
	}
	oid, err := db.GetOID(wantedOID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, apiError{err.Error()})
	}
	return ctx.JSON(http.StatusOK, oid)
}

func getOIDRelation(ctx echo.Context) error {
	wantedOID := ctx.Param("oid")
	db, err := database.GetDB()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, apiError{errors.Wrap(err, "failed to get db").Error()})
	}
	oids, err := db.GetOIDRelation(wantedOID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, apiError{err.Error()})
	}
	return ctx.JSON(http.StatusOK, oids)
}

func getOIDParent(ctx echo.Context) error {
	wantedOID := ctx.Param("oid")
	db, err := database.GetDB()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, apiError{errors.Wrap(err, "failed to get db").Error()})
	}
	oids, err := db.GetOIDParent(wantedOID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, apiError{err.Error()})
	}
	return ctx.JSON(http.StatusOK, oids)
}

func getOIDSiblings(ctx echo.Context) error {
	wantedOID := ctx.Param("oid")
	db, err := database.GetDB()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, apiError{errors.Wrap(err, "failed to get db").Error()})
	}
	oids, err := db.GetOIDSiblings(wantedOID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, apiError{err.Error()})
	}
	if len(oids) == 0 {
		return ctx.JSON(http.StatusNotFound, apiError{"no result"})
	}
	return ctx.JSON(http.StatusOK, oids)
}

func getOIDChildren(ctx echo.Context) error {
	wantedOID := ctx.Param("oid")
	db, err := database.GetDB()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, apiError{errors.Wrap(err, "failed to get db").Error()})
	}
	oids, err := db.GetOIDChildren(wantedOID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, apiError{err.Error()})
	}
	if len(oids) == 0 {
		return ctx.JSON(http.StatusNotFound, apiError{"no result"})
	}
	return ctx.JSON(http.StatusOK, oids)
}
