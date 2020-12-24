package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
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
	"github.com/ivan3bx/pickaxx/minecraft"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type processHandler struct {
	logFile *os.File
	manager pickaxx.ProcessManager
	writer  io.Writer
}

// newlineWriter is a writer that inserts '\n' newlines after each call.
type newlineWriter struct {
	wrapped io.Writer
}

func (w *newlineWriter) Write(p []byte) (n int, err error) {
	if n, err := w.wrapped.Write(p); err != nil {
		return n, err
	}
	return w.wrapped.Write([]byte("\n"))
}

// monitor output coming from a process by sending it where it needs to go.
func (h *processHandler) monitor(ch <-chan pickaxx.Data) error {
	var (
		err error
	)

	if h.logFile != nil {
		log.Warn("processHandler already monitoring activity?")
	}

	// set up log file
	if h.logFile, err = ioutil.TempFile(os.TempDir(), fmt.Sprintf("pickaxx_%d", minecraft.DefaultPort)); err != nil {
		return err
	}

	// create a new routine to funnel output where it needs to go
	go func() {
		enc := json.NewEncoder(h.writer)
		w := &newlineWriter{h.logFile}

		for newData := range ch {
			if val, ok := newData.(pickaxx.ConsoleData); ok {
				io.WriteString(w, val.String())
			}
			enc.Encode(newData)
		}
	}()

	return nil
}

func (h *processHandler) rootHandler(c *gin.Context) {
	var (
		manager = h.manager
		lines   []string
		status  string
	)

	if manager.Running() {
		// set a status
		status = "Running"

		// set recent activity
		content, _ := ioutil.ReadFile(h.logFile.Name())
		lines = strings.Split(string(content), "\n")
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

func (h *processHandler) startServerHandler(c *gin.Context) {
	var (
		manager = h.manager
	)

	activity, err := manager.Start()

	if err != nil {
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

	h.monitor(activity)
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
