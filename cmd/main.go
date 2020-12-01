package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/gin-gonic/gin"
	"github.com/ivan3bx/pickaxx"
)

func main() {
	var (
		clientMgr  pickaxx.ClientManager = pickaxx.ClientManager{}
		processMgr pickaxx.ServerManager = pickaxx.NewServerManager(&clientMgr)
	)

	log.SetLevel(log.DebugLevel)
	log.SetHandler(cli.Default)

	// start web server
	e := gin.New()
	e.Use(gin.Logger(), gin.Recovery())

	e.Static("/assets", "public")
	e.LoadHTMLFiles("templates/index.html")

	e.GET("/ws", webSocketHandler(&clientMgr))

	routes := e.Group("/", managerMiddleware(processMgr))
	{
		routes.GET("/", rootHandler)
		routes.POST("/start", startServerHandler)
		routes.POST("/stop", stopServerHandler)
	}

	srv := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: e,
	}

	// Run the server
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server failed: %w", err)
		}
	}()

	// shutdown on interrupt
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	shutdown(srv, processMgr)
}

func shutdown(srv *http.Server, manager pickaxx.ServerManager) {
	log.Debug("shutdown initiated")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	manager.StopServer(ctx)
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Error("server failed to shutdown")
	}

	log.Info("shutdown complete")
	os.Exit(1)
}
