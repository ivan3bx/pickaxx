package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"time"

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

// NewProcessHandler returns a new process handler.
func NewProcessHandler(cm *pickaxx.ClientManager) *ProcessHandler {
	ph := ProcessHandler{
		active:       make(map[string]*managedServer),
		clientWriter: cm,
	}

	servers := make(map[string]*managedServer)
	dirNames, _ := ioutil.ReadDir(dataDir)
	for _, srvName := range dirNames {
		key := srvName.Name()
		mapping := &managedServer{
			ProcessManager: minecraft.New(filepath.Join(dataDir, srvName.Name()), minecraft.DefaultPort),
			Reporter:       &loggingReporter{writer: cm},
		}
		servers[key] = mapping
	}

	ph.active = servers

	return &ph
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

func (h *ProcessHandler) index(c *gin.Context) {
	var (
		key string
	)
	serverList := serverList(dataDir)
	if len(serverList) > 0 {
		key = serverList[0].Key
	}

	renderTemplate(c, "index.html", gin.H{
		"logLines": []string{},
		"status":   "Unknown",
		"servers":  serverList,
		"selected": key,
	})
}

func (h *ProcessHandler) newServer(c *gin.Context) {
	renderTemplate(c, "index.html", gin.H{
		"logLines": []string{},
		"status":   "Unknown",
		"servers":  serverList(dataDir),
		"selected": "_default", // TODO: none should be selected
	})

}

func (h *ProcessHandler) show(c *gin.Context) {
	var (
		lines  []string
		status string
	)

	if resolveKey(&c.Params) == "new" {
		h.newServer(c)
		return
	}

	m, err := h.resolveServer(c)

	if err == nil && m.Running() {
		status = "Running"                 // set a status
		lines = m.Reporter.ConsoleOutput() // set recent activity
	}

	renderTemplate(c, "index.html", gin.H{
		"logLines": lines,
		"status":   status,
		"servers":  serverList(dataDir),
		"selected": resolveKey(&c.Params),
	})
}

func (h *ProcessHandler) start(c *gin.Context) {

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

func (h *ProcessHandler) create(c *gin.Context) {
	var (
		file    *multipart.FileHeader
		tempDir string
		err     error
	)
	log := log.WithField("action", "createNew()")

	if file, err = c.FormFile("file"); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "file not received"})
		return
	}

	if file.Header.Get("Content-Type") != "application/java-archive" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "unsupported file type"})
		return
	}

	if tempDir, err = ioutil.TempDir(dataDir, ".new_server_*.tmp"); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"err": "unable to create temp directory"})
		return
	}

	if err := c.SaveUploadedFile(file, filepath.Join(tempDir, "server.jar")); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"err": "unable to save file"})
		return
	}

	absPath, _ := filepath.Abs(tempDir)
	log.WithField("dir", absPath).Info("staged new server")

	placeholder := fmt.Sprintf("server #%d", fileCount(dataDir))

	c.JSON(http.StatusOK, gin.H{
		"output": "file is staged",
		"key":    filepath.Base(tempDir),
		"name":   placeholder,
	})
}

// ServerMetadata is serialized description of a minecraft server.
type ServerMetadata struct {
	Key       string    `form:"key"  json:"key"`
	Name      string    `form:"name" json:"name"`
	CreatedAt time.Time `form:"-"    json:"createdAt"`
}

func (h *ProcessHandler) commit(c *gin.Context) {
	var (
		reqBody ServerMetadata
	)

	log := log.WithField("action", "commitNew()")

	if err := c.ShouldBind(&reqBody); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "invalid request"})
		return
	}

	if reqBody.Key == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "key is required"})
		return
	}

	if reqBody.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "name is required"})
		return
	}

	reqBody.CreatedAt = time.Now()

	// resolve staged directory
	oldPath := filepath.Join(dataDir, reqBody.Key)
	if _, err := os.Stat(oldPath); err != nil {
		log.WithError(err).Error("old path not found")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"err": "staged server not found. unable to continue.",
		})
		return
	}

	// update key

	// write out metadata to staged dir
	reqBody.Key = ""
	jsonOut, _ := json.MarshalIndent(&reqBody, "", "  ")
	if err := ioutil.WriteFile(filepath.Join(oldPath, "pickaxx.json"), jsonOut, 0644); err != nil {
		log.WithError(err).Error("failed to write metadata")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"err": "failed to write server metadata. unable to continue.",
		})
		return
	}

	// resolve permanent home for staged directory
	newPath := filepath.Join(dataDir, sanitize(reqBody.Name))
	if _, err := os.Stat(newPath); err == nil {
		log.WithField("file", newPath).Error("new path was found ; name conflict!")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"err": "name conflicts with existing server",
		})
		return
	}

	// rename oldPath -> newPath
	if err := os.Rename(oldPath, newPath); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"err": "unable to commit change",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": fmt.Sprintf("/server/%s", sanitize(reqBody.Name)),
	})
}

func (h *ProcessHandler) delete(c *gin.Context) {
	var (
		key string
		err error
	)

	if key = c.PostForm("key"); key == "" {
		log.Error("key is blank")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "no server found"})
		return
	}

	stagingDir, err := findStagedServerPath(dataDir, key)
	stagingDir = filepath.Join(dataDir, stagingDir)

	if err != nil {
		log.WithField("key", key).Error("key does not point to a staged server")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": "invalid key"})
		return
	}

	log.Debugf("removing staging directory.. %s", stagingDir)

	if err := os.Remove(filepath.Join(stagingDir, "server.jar")); err != nil {
		log.WithError(err).Warn("server.jar missing")
	}

	if err := os.Remove(stagingDir); err != nil {
		log.WithError(err).Error("error on delete")
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"err": "unable to clear staged directory"})
	}
}

func (h *ProcessHandler) stop(c *gin.Context) {

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
	return sanitize(key)
}

func sanitize(str string) string {
	str = strings.ReplaceAll(str, " ", "_")
	reg := regexp.MustCompile("[^a-zA-Z0-9_]+")
	str = reg.ReplaceAllString(str, "")

	if str == "" {
		str = "_default"
	}

	return str
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

func fileCount(path string) int {
	files, _ := ioutil.ReadDir(path)
	return len(files)
}

func findStagedServerPath(path, name string) (string, error) {
	const nullPath = "._" // return a sentinel value instead of empty path

	// strip any path prefix from name
	if name != filepath.Base(name) {
		return nullPath, errors.New("name can not contain path")
	}

	files, err := ioutil.ReadDir(path)

	if err != nil {
		return nullPath, err
	}

	for _, f := range files {
		if !f.IsDir() || !strings.HasPrefix(f.Name(), ".new_server_") {
			continue
		}

		if filepath.Base(f.Name()) == name {
			return f.Name(), nil
		}
	}

	return nullPath, errors.New("not found")
}

func renderTemplate(c *gin.Context, name string, params map[string]interface{}) {
	t := template.New(name)

	for _, tf := range tmpls.List() {
		tmpl, _ := tmpls.FindString(tf)
		t.Parse(tmpl)
	}

	if err := t.ExecuteTemplate(c.Writer, name, params); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func serverList(dataDir string) []*ServerMetadata {
	serverList := []*ServerMetadata{}
	dirs, _ := ioutil.ReadDir(dataDir)
	for _, entry := range dirs {
		if entry.IsDir() {
			metadataPath := filepath.Join(dataDir, entry.Name(), "pickaxx.json")
			metadataFile, err := os.Open(metadataPath)
			if err != nil {
				log.WithField("path", metadataPath).Warn("unable to find server metadata file")
				continue
			}
			var meta ServerMetadata
			json.NewDecoder(metadataFile).Decode(&meta)
			meta.Key = entry.Name()
			serverList = append(serverList, &meta)
		}
	}
	return serverList
}
