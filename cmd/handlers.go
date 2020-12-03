package main

import (
	"errors"
	"io"

	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ivan3bx/pickaxx"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type processHandler struct {
	manager *pickaxx.ProcessManager
	writer  io.Writer
}

func (h *processHandler) rootHandler(c *gin.Context) {
	var (
		manager = h.manager
		lines   []string
	)

	if manager.Running() {
		content, _ := ioutil.ReadFile("testserver/logs/latest.log")
		lines = strings.Split(string(content), "\n")
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"logLines": lines,
		"status":   manager.CurrentState().String(),
	})
}

func (h *processHandler) startServerHandler(c *gin.Context) {
	var (
		manager = h.manager
		w       = h.writer
	)

	if err := manager.Start(w); err != nil {
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

func (h *processHandler) stopServerHandler(c *gin.Context) {
	var (
		manager = h.manager
	)

	if err := manager.Stop(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
	}
}

type clientHandler struct {
	manager *pickaxx.ClientManager
}

func (h *clientHandler) webSocketHandler(c *gin.Context) {
	var (
		cm   = h.manager
		conn *websocket.Conn
		err  error
	)

	if conn, err = upgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	cm.AddClient(conn)
}
