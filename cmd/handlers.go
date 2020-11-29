package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ivan3bx/pickaxx"
)

// ErrSystem reflects za non-recoverable system error
var ErrSystem = errors.New("system error")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func rootHandler(c *gin.Context) {
	var (
		lines []string
	)

	manager := getServerManager(c)

	if manager.Active() {
		content, _ := ioutil.ReadFile("testserver/logs/latest.log")
		lines = strings.Split(string(content), "\n")
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"logLines": lines,
		"status":   pickaxx.Unknown,
	})
}

func webSocketHandler(cm *pickaxx.ClientManager) func(c *gin.Context) {
	return func(c *gin.Context) {
		var (
			conn *websocket.Conn
			err  error
		)

		if conn, err = upgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		cm.AddClient(conn)
	}
}

func startServerHandler(c *gin.Context) {
	manager := getServerManager(c)

	if err := manager.StartServer(); err != nil {
		var status int
		var message string

		if errors.Is(err, pickaxx.ErrProcessExists) {
			status = http.StatusBadRequest
			message = "server already running"
		} else {
			status = http.StatusInternalServerError
			message = "failed to start server"
		}

		c.AbortWithStatusJSON(status, gin.H{"err": message})
	}
}

func stopServerHandler(c *gin.Context) {
	manager := getServerManager(c)

	if err := manager.StopServer(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
	}
}
