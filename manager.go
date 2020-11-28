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

// ErrProcessExists exists when a new server process can not be started.
var ErrProcessExists = errors.New("unable to start new process")

// ErrNoProcess occurs when no process exists to take an action on
var ErrNoProcess = errors.New("no process running")

// Manager manages one or more Minecraft servers.
type Manager interface {
	Active() bool
	Run()
	AddClient(*websocket.Conn)
	StartServer() error
	StopServer() error
}

// NewServerManager creates a new instance of a server manager.
func NewServerManager() Manager {
	return &serverManager{
		state: ManagedState{
			current: Unknown,
			update:  make(chan ServerState),
		},
		output:     make(chan []byte, 10),
		register:   make(chan *client),
		unregister: make(chan *client),
		clients:    make(map[*client]bool),
	}
}

type serverManager struct {
	state ManagedState
	// stop       chan bool
	// status     chan string
	output     chan []byte
	register   chan *client
	unregister chan *client

	killServerFunc context.CancelFunc

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

// TODO put this in a goroutine; poll status every 500ms and send
// a state update through state manager if process died.
func (m *serverManager) Active() bool {
	return m.process != nil && m.process.Signal(syscall.Signal(0)) == nil
}

func (m *serverManager) StartServer() error {
	if m.state.current == Started {
		return fmt.Errorf("server already running: %w", ErrProcessExists)
	}

	m.state.update <- Starting

	ctx := log.NewContext(context.Background(), log.WithField("action", "runloop"))
	ctx, cancel := context.WithCancel(ctx)
	log := log.FromContext(ctx)

	cmd := exec.CommandContext(ctx, "java", MaxMem, MinMem, "-jar", JarFile, "nogui")
	cmd.Dir = "testserver"

	m.killServerFunc = cancel

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

	m.state.update <- Started

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

func (m *serverManager) StopServer() error {
	if !m.Active() {
		log.Warn("Server not running. No process found.")
		return ErrNoProcess
	}

	m.state.update <- Stopping

	// set timer to force shutdown if clean shutdown stalls
	go func() {
		timer := time.NewTimer(time.Second * 10)

		select {
		case <-timer.C:
			if m.state.current != Stopped {
				log.Debug("timer expired. stopping server.")
				m.killServerFunc()
				m.process = nil
				m.state.update <- Stopped
			}
		}
	}()

	// attempt clean shutdown
	go func() {
		m.console.Write([]byte("/stop\n"))
		m.process.Wait()
		m.killServerFunc()
		m.process = nil
		m.state.update <- Stopped
	}()

	return nil
}

func (m *serverManager) Run() {
	for {
		clients := m.clients

		select {
		case client := <-m.register:
			clients.Add(client)
		case client := <-m.unregister:
			clients.Remove(client)
		case status := <-m.state.update:
			m.state.Lock()
			m.state.current = status
			m.state.Unlock()

			clients.broadcast(gin.H{
				"status": status.String(),
			})
		case output := <-m.output:
			clients.broadcast(gin.H{
				"output": string(output),
			})
		}
	}
}
