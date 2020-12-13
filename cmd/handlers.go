package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"io/ioutil"
	"net/http"
	"strings"

	"github.com/apex/log"
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
		content, _ := ioutil.ReadFile(manager.Logfile())
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

func (h *processHandler) createServerHandler(c *gin.Context) {
	var (
		tempFile *os.File
		err      error
	)

	file, err := c.FormFile("file")

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "file not received"})
		return
	}

	if file.Header.Get("Content-Type") != "application/java-archive" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "unsupported file type"})
		return
	}

	ext := filepath.Ext(file.Filename)
	filename := strings.TrimSuffix(file.Filename, ext)

	if tempFile, err = ioutil.TempFile("", fmt.Sprintf("%s-*%s", filename, ext)); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"err": "unable to write save file"})
		return
	}

	if err := c.SaveUploadedFile(file, tempFile.Name()); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"err": "unable to save file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output": "file is staged",
		"key":    filepath.Base(tempFile.Name()),
	})
}

func (h *processHandler) stopServerHandler(c *gin.Context) {
	var (
		manager = h.manager
	)

	if err := manager.Stop(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
	}
}

func (h *processHandler) sendHandler(c *gin.Context) {
	var (
		manager = h.manager
	)

	var data = map[string]string{}
	if err := c.BindJSON(&data); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	cmd := data["command"]

	if !manager.Running() {
		h.writer.Write([]byte("Server not running. Unable to respond to commands."))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"output": "Server not running."})
		return
	}

	log.WithField("cmd", cmd).Info("executing command")

	if err := manager.Submit(cmd); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"output": "error submitting command"})
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
