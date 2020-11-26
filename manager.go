package pickaxx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/apex/log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	// MaxMem is the maximum allocated memory
	MaxMem = "-Xmx1024m"

	// MinMem is the minimum allocated memory
	MinMem = "-Xms512m"

	// JarFile is the name of the server jar as it exists on disk
	JarFile = "server.jar"
)

// ErrProcessStart exists when a new server process can not be started.
var ErrProcessStart = errors.New("unable to start new process")

// Manager manages one or more Minecraft servers.
type Manager interface {
	Active() bool
	Run()
	AddClient(*websocket.Conn)
	StartServer() error
	StopServer()
}

// NewServerManager creates a new instance of a server manager.
func NewServerManager() Manager {
	return &serverManager{
		status:     make(chan string),
		output:     make(chan []byte, 10),
		register:   make(chan *client),
		unregister: make(chan *client),
		clients:    make(map[*client]bool),

		// TODO: this channel should be cleared on server startup or shutdown
		// gets invoked if 'stop' button pressed before 'start'
		stop: make(chan bool),
	}
}

type serverManager struct {
	stop       chan bool
	status     chan string
	output     chan []byte
	register   chan *client
	unregister chan *client

	clients clients
	console io.Writer
	process *os.Process
}

func (m *serverManager) AddClient(conn *websocket.Conn) {
	cl := &client{
		manager: m,
		conn:    conn,
	}

	m.register <- cl
}

func startShutdownRoutine(ctx context.Context, cancel context.CancelFunc, m *serverManager) {
	log := log.FromContext(ctx).WithField("action", "cancelRoutine")

	select {
	case <-ctx.Done():
		log.Debug("context already marked done")
	case <-m.stop:
		go cancelWithDeadline(ctx, cancel, time.Second*10)
		m.console.Write([]byte("/stop\n"))
		m.process.Wait()
		cancel()
	}
}

func cancelWithDeadline(ctx context.Context, cancel context.CancelFunc, limit time.Duration) {
	log := log.FromContext(ctx).WithField("action", "cancelWithDeadline")
	timer := time.NewTimer(limit)

	select {
	case <-timer.C:
		log.Debug("timer expired")
		cancel()
	case <-ctx.Done():
		log.Debug("no action needed")
	}
}

func (m *serverManager) Active() bool {
	return m.process != nil && m.process.Signal(syscall.Signal(0)) == nil
}

func (m *serverManager) StartServer() error {
	m.status <- "starting"

	if m.Active() {
		return fmt.Errorf("server already running: %w", ErrProcessStart)
	}

	ctx := log.NewContext(context.Background(), log.WithField("action", "runloop"))
	ctx, cancel := context.WithCancel(ctx)
	log := log.FromContext(ctx)

	cmd := exec.CommandContext(ctx, "java", MaxMem, MinMem, "-jar", JarFile, "nogui")
	cmd.Dir = "testserver"

	go startShutdownRoutine(ctx, cancel, m)

	var (
		cmdIn  io.Writer
		cmdOut io.Reader
		cmdErr io.Reader

		err error
	)

	if cmdIn, err = cmd.StdinPipe(); err != nil {
		return err
	}

	if cmdOut, err = cmd.StdoutPipe(); err != nil {
		return err
	}

	if cmdErr, err = cmd.StderrPipe(); err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		log.WithError(err).Error("failed to start command")
		return err
	}

	m.process = cmd.Process
	m.console = cmdIn

	go m.captureOutput(cmdOut)
	go m.captureOutput(cmdErr)

	m.status <- "running"
	return nil
}

func (m *serverManager) captureOutput(src io.Reader) {
	buf := make([]byte, 1024, 1024)

	for {
		n, err := src.Read(buf[:])
		if n > 0 {
			data := buf[:n]
			m.output <- data
		}
		if err != nil {
			if err == io.EOF {
				return
			}

			log.WithError(err).Error("read/write failed")
			return
		}
	}
}

func (m *serverManager) StopServer() {
	if m.process == nil {
		log.Warn("Server not running. No process found.")
		return
	}

	m.status <- "stopping"
	m.stop <- true
}

func (m *serverManager) Run() {
	for {
		clients := m.clients

		select {
		case client := <-m.register:
			clients.Add(client)
		case client := <-m.unregister:
			clients.Remove(client)
		case status := <-m.status:
			clients.broadcast(gin.H{
				"status": status,
			})
		case output := <-m.output:
			clients.broadcast(gin.H{
				"output": string(output),
			})
		}
	}
}
