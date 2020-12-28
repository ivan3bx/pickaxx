package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
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

// ProcessHandler exposes handlers for the lifecycle of process managers.
type ProcessHandler struct {
	procs        map[string]pickaxx.ProcessManager
	trackers     map[string]pickaxx.Logger
	clientWriter io.Writer
}

// NewProcessHandler creates a new process handler ready to handle requests.
func NewProcessHandler(writer io.Writer) *ProcessHandler {
	return &ProcessHandler{
		procs:        make(map[string]pickaxx.ProcessManager),
		trackers:     make(map[string]pickaxx.Logger),
		clientWriter: writer,
	}
}

// Stop will reset this handler's state, and signal to any ProcessManager's associated
// with this handler to stop processing immediately.
func (h *ProcessHandler) Stop() {
	for _, manager := range h.procs {
		manager.Stop()
	}
	h.procs = make(map[string]pickaxx.ProcessManager)
	h.trackers = make(map[string]pickaxx.Logger)
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

func (h *ProcessHandler) rootHandler(c *gin.Context) {
	var (
		key    string
		lines  []string
		status string
	)

	key = resolveKey(&c.Params)

	if srv := h.procs[key]; srv != nil && srv.Running() {
		// set a status
		status = "Running"

		// set recent activity
		lines = h.trackers[key].History(-1)
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
	var (
		activity <-chan pickaxx.Data
		manager  pickaxx.ProcessManager
		tracker  pickaxx.Logger
		err      error
	)

	key := resolveKey(&c.Params)

	if m := h.procs[key]; m != nil && m.Running() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "server already running"})
		return
	}

	manager = minecraft.New(minecraft.DefaultPort)
	h.procs[key] = manager

	tracker = minecraft.NewTracker(h.clientWriter)
	h.trackers[key] = tracker

	if activity, err = manager.Start(); err != nil {
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
		return
	}

	go tracker.Track(activity)
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
	var (
		manager pickaxx.ProcessManager
	)

	key := resolveKey(&c.Params)

	if manager = h.procs[key]; manager == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "server is not running"})
		return
	}

	if err := manager.Stop(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
	}
}

func (h *ProcessHandler) sendCommand(c *gin.Context) {
	var (
		manager pickaxx.ProcessManager
	)

	key := resolveKey(&c.Params)

	if manager = h.procs[key]; manager == nil {
		h.clientWriter.Write([]byte("Server not running. Unable to respond to commands."))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "server is not running"})
		return
	}

	var data = map[string]string{}
	if err := c.BindJSON(&data); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	cmd := data["command"]

	log.WithField("cmd", cmd).Info("executing command")

	if err := manager.Submit(cmd); err != nil {
		h.clientWriter.Write([]byte("Server not running. Unable to respond to commands."))
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
