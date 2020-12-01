package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/ivan3bx/pickaxx"
)

func main() {
	var (
		clientMgr  *pickaxx.ClientManager  = &pickaxx.ClientManager{}
		processMgr *pickaxx.ProcessManager = &pickaxx.ProcessManager{}
	)

	configureLogging(log.DebugLevel)

	e := newRouter()

	// routes: process handling
	ph := processHandler{
		manager: processMgr,
		writer:  clientMgr,
	}
	{
		e.GET("/", ph.rootHandler)
		e.POST("/start", ph.startServerHandler)
		e.POST("/stop", ph.stopServerHandler)

	}

	// routes: client handling
	ch := clientHandler{clientMgr}
	{
		e.GET("/ws", ch.webSocketHandler)
	}

	// Start the web server
	srv := startWebServer(e)

	// shutdown on interrupt
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Debug("shutdown initiated")
	{
		stopWebServer(srv)
		stopProcesses(processMgr)
	}
	log.Info("shutdown complete")
}

func configureLogging(level log.Level) {
	log.SetLevel(level)
	log.SetHandler(cli.Default)
}

func stopProcesses(m *pickaxx.ProcessManager) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m.Stop(ctx)
}
