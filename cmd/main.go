package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/gin-gonic/gin"
	"github.com/ivan3bx/pickaxx"
)

func main() {

	configureLogging(log.DebugLevel)

	e := newRouter()

	// routes: process handling
	ph := ProcessHandler{
		active:       make(map[string]*managedServer),
		clientWriter: &pickaxx.ClientManager{},
	}

	e.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/server/_default") })

	srvRoute := e.Group("/server")
	{
		srvRoute.GET("/:key", ph.rootHandler)
		srvRoute.POST("/", ph.createNew)
		srvRoute.POST("/:key/start", ph.startServer)
		srvRoute.POST("/:key/stop", ph.stopServer)
		srvRoute.POST("/:key/send", ph.sendCommand)

	}

	// routes: client handling
	e.GET("/ws", webSocketHandler(ph.clientWriter.AddClient))

	// Start the web server
	srv := startWebServer(e)

	// shutdown on interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Debug("shutdown initiated")
	{
		stopWebServer(srv)
		ph.Stop()
	}
	log.Info("shutdown complete")
}

func configureLogging(level log.Level) {
	log.SetLevel(level)
	log.SetHandler(cli.Default)
}
