package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/gin-gonic/gin"
	"github.com/ivan3bx/pickaxx"
)

const defaultLogLevel = log.InfoLevel

var (
	dataDir string
	verbose bool
	version string // version is only set on release
)

func init() {
	flag.StringVar(&dataDir, "d", "./pxdata", "directory to store server data")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
}

func main() {
	flag.Parse()

	configureLogging()
	configureDataDirectory()

	if version != "" {
		log.Infof("Pickaxx: %s ⛏", version)
	}

	e := newRouter()

	// routes: process handling
	clientManager := pickaxx.ClientManager{}
	ph := NewProcessHandler(&clientManager)

	e.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/server/_default") })
	e.GET("/server/:key", ph.indexList)
	e.POST("/server", ph.createNew)
	e.PUT("/server", ph.commitNew)
	e.DELETE("/server", ph.cancelNew)
	e.POST("/server/:key/start", ph.startServer)
	e.POST("/server/:key/stop", ph.stopServer)
	e.POST("/server/:key/send", ph.sendCommand)

	// routes: client handling
	e.GET("/ws", webSocketHandler(clientManager.AddClient))

	// Start the web server
	srv := startWebServer(e)
	log.Infof("Server running at: http://%s", srv.Addr)

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

func configureLogging() {
	level := defaultLogLevel

	if verbose {
		level = log.DebugLevel
	}

	log.SetLevel(level)
	log.SetHandler(cli.Default)
}

func configureDataDirectory() {
	fmt.Println("Making dir:", dataDir)
	err := os.Mkdir(dataDir, 0755)
	if err != nil && os.IsNotExist(err) {
		log.WithError(err).Fatalf("unable to create directory: %s", dataDir)
	}
}
