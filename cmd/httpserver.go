package main

import (
	"context"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr/v2"
)

var (
	assets = packr.New("assets", "../public")
	tmpls  = packr.New("templates", "../templates")
)

func newRouter() *gin.Engine {
	if version != "" {
		gin.SetMode(gin.ReleaseMode)
	}

	e := gin.New()
	e.Use(gin.Logger(), gin.Recovery())

	e.StaticFS("/assets", assets)
	return e
}

func startWebServer(e http.Handler) *http.Server {
	srv := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: e,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server failed: %w", err)
		}
	}()

	return srv
}

func stopWebServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Error("server failed to shutdown")
	}
}
