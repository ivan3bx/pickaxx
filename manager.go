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

// ErrNoProcess occurs when no process exists to take an action on.
var ErrNoProcess = errors.New("no process running")

// ErrInvalidClient occurs when a client is not valid.
var ErrInvalidClient = errors.New("client not valid")

// ServerManager manages an instance of Minecraft server.
type ServerManager interface {
	Active() bool
	StartServer() error
	StopServer(context.Context) error
}

// NewServerManager creates a new instance of a server manager, and sends all output to the given writer.
func NewServerManager(w io.Writer) ServerManager {
	return &serverManager{
		state:  Unknown,
		writer: w,
	}
}

type serverManager struct {
	state          ServerState
	console        io.Writer
	process        *os.Process
	writer         io.Writer
	killServerFunc context.CancelFunc
}

// TODO put this in a goroutine; poll status every 500ms and send
// a state update through state manager if process died.
func (m *serverManager) Active() bool {
	return m.process != nil && m.process.Signal(syscall.Signal(0)) == nil
}

func (m *serverManager) StartServer() error {
	if m.state == Started {
		return fmt.Errorf("server already running: %w", ErrProcessExists)
	}

	m.updateState(Starting)

	ctx := log.NewContext(context.Background(), log.WithField("action", "startServer"))
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

	m.updateState(Started)

	return nil
}

func (m *serverManager) captureOutput(src io.Reader) {
	buf := make([]byte, 1024, 1024)

	for {
		n, err := src.Read(buf[:])
		if n > 0 {
			data := buf[:n]
			m.writer.Write(data)
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

func (m *serverManager) StopServer(ctx context.Context) error {
	if !m.Active() {
		return ErrNoProcess
	}

	m.updateState(Stopping)

	// set timer to force shutdown if clean shutdown stalls
	go func() {
		timer := time.NewTimer(time.Second * 10)

		select {
		case <-ctx.Done():
			log.Debug("context marked as done")
		case <-timer.C:
			log.Debug("timer deadline expired")
		}

		m.killServerFunc()
		m.process = nil
		m.updateState(Stopped)
	}()

	log.Info("clean shutdown starting..")
	m.console.Write([]byte("/stop\n"))
	if _, err := m.process.Wait(); err != nil {
		log.WithError(err).Warn("clean shutdown failed with error")
	}
	m.process = nil
	m.updateState(Stopped)
	log.Info("clean shutdown completed")

	return nil
}

func (m *serverManager) updateState(newState ServerState) {
	m.state = newState
	jsonString := fmt.Sprintf(`{"status":"%s"}`, newState.String())
	m.writer.Write([]byte(jsonString))
}
