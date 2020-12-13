package main

import (
	"os"
	"os/signal"
	"syscall"

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
		e.POST("/server", ph.createServerHandler)
		e.POST("/send", ph.sendHandler)

	}

	// routes: client handling
	ch := clientHandler{clientMgr}
	{
		e.GET("/ws", ch.webSocketHandler)
	}

	// Start the web server
	srv := startWebServer(e)

	// shutdown on interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Debug("shutdown initiated")
	{
		stopWebServer(srv)
		stopProcesses(processMgr)
		stopClientManager(clientMgr)
	}
	log.Info("shutdown complete")
}

func configureLogging(level log.Level) {
	log.SetLevel(level)
	log.SetHandler(cli.Default)
}

func stopProcesses(m *pickaxx.ProcessManager) {
	m.Stop()
}

func stopClientManager(cl *pickaxx.ClientManager) {
	cl.Close()
}
