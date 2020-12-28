package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"

	"io/ioutil"
	"net/http"
	"strings"

	"github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ivan3bx/pickaxx"
	"github.com/ivan3bx/pickaxx/minecraft"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type managedServer struct {
	pickaxx.ProcessManager
	Reporter *loggingReporter
}

// ProcessHandler exposes handlers for the lifecycle of process managers.
type ProcessHandler struct {
	active       map[string]*managedServer
	clientWriter *pickaxx.ClientManager
}

// Stop will reset this handler's state, and signal to any ProcessManager's associated
// with this handler to stop processing immediately.
func (h *ProcessHandler) Stop() {
	for key, ms := range h.active {
		ms.Stop()
		delete(h.active, key)
	}
	h.clientWriter.Close()
}

func (h *ProcessHandler) rootHandler(c *gin.Context) {
	var (
		lines  []string
		status string
	)

	if m, err := h.resolveServer(c); err == nil && m.Running() {
		status = "Running"                 // set a status
		lines = m.Reporter.ConsoleOutput() // set recent activity
	}

	html, err := tmpls.FindString("index.html")

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	t := template.New("")
	t.Parse(html)

	err = t.ExecuteTemplate(c.Writer, "", gin.H{
		"logLines": lines,
		"status":   status,
	})

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func (h *ProcessHandler) startServer(c *gin.Context) {

	manager, err := h.resolveServer(c)

	if err == nil && manager.Running() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "server already running"})
		return
	}

	server := minecraft.New(dataDir, minecraft.DefaultPort)
	reporter := &loggingReporter{writer: h.clientWriter}

	// start the server
	activity, err := server.Start()

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	go reporter.Report(activity)

	key := resolveKey(&c.Params)
	h.active[key] = &managedServer{
		ProcessManager: server,
		Reporter:       reporter,
	}
}

func (h *ProcessHandler) createNew(c *gin.Context) {
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

func (h *ProcessHandler) stopServer(c *gin.Context) {

	manager, err := h.resolveServer(c)

	if err != nil || !manager.Running() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "server is not running"})
		return
	}

	if err := manager.Stop(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
	}

}

func (h *ProcessHandler) sendCommand(c *gin.Context) {

	manager, err := h.resolveServer(c)

	if err != nil {
		h.clientWriter.Write(rawData{"Server not running. Unable to respond to commands."})
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "server is not running"})
		return
	}

	var commandData = map[string]string{}
	if err := c.BindJSON(&commandData); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	cmd := commandData["command"]

	log.WithField("cmd", cmd).Info("executing command")

	if err := manager.Submit(cmd); err != nil {
		h.clientWriter.Write(rawData{"Server not running. Unable to respond to commands."})
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"output": "error submitting command"})
	}
}

// webSocketHandler returns a handler for upgrading websocket connections, and takes a
// function that will receive any new, upgraded connections.
func webSocketHandler(newConnFunc func(*websocket.Conn)) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		newConnFunc(conn)
	}
}

func resolveKey(p *gin.Params) string {
	key := p.ByName("key")

	key = strings.ReplaceAll(key, " ", "_")
	reg := regexp.MustCompile("[^a-zA-Z0-9_]+")
	key = reg.ReplaceAllString(key, "")

	if key == "" {
		key = "_default"
	}

	return key
}

func (h *ProcessHandler) resolveServer(c *gin.Context) (*managedServer, error) {
	key := resolveKey(&c.Params)
	log := log.WithField("key", key)

	if h.active == nil {
		h.active = make(map[string]*managedServer)
	}
	if m := h.active[key]; m != nil {
		log.WithField("running", m.ProcessManager.Running()).Debug("resolved server")
		return m, nil
	}

	log.Debug("no server found")
	return nil, errors.New("server not found")
}

type rawData struct {
	Value string
}

func (d rawData) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"output": d.Value})
}
