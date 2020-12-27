package minecraft

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ivan3bx/pickaxx"
)

const (
	// MaxMem is the maximum allocated memory
	MaxMem = "-Xmx1024m"

	// MinMem is the minimum allocated memory
	MinMem = "-Xms512m"

	// JarFile is the name of the server jar as it exists on disk
	JarFile = "server.jar"

	// DefaultPort is the default minecraft server port
	DefaultPort = 25565

	// DefaultWorkingDir is the default working directory
	DefaultWorkingDir = "testserver"
)

// DefaultCommand is the name of the executable.
var DefaultCommand = []string{"java", MaxMem, MinMem, "-jar", JarFile, "nogui"}

// ErrNoProcess signifies no process exists to take an action on.
var ErrNoProcess = errors.New("no process running")

// New creates a new process manager for an instance of Minecraft server.
func New(port int) pickaxx.ProcessManager {
	return &serverManager{
		Port: port,
	}
}

// serverManager manages the Minecraft server's process lifecycle.
type serverManager struct {
	Command    []string // Defaults to 'DefaultExecArgs' if not set.
	WorkingDir string   // Defaults to 'DefaultWorkignDir' if not set.
	Port       int      // Server port for Minecraft server instance.

	// Child process
	cmd    *exec.Cmd
	cmdIn  io.Writer
	cmdOut io.Reader

	// state transition
	state     ServerState
	lock      sync.RWMutex
	nextState chan ServerState

	// observers of state transitions
	notifier StatusNotifier
}

// Start will initialize a new process, sending all output to the provided
// channel and set values on this object to track process state.
// This returns an error if the process is already running.
func (m *serverManager) Start() (<-chan pickaxx.Data, error) {
	if m.Running() {
		return nil, fmt.Errorf("server already running: %w", pickaxx.ErrProcessExists)
	}

	// initialize
	if len(m.Command) == 0 {
		m.Command = DefaultCommand
	}

	if m.Port == 0 {
		m.Port = DefaultPort
	}

	if m.WorkingDir == "" {
		m.WorkingDir = DefaultWorkingDir
	}

	if _, err := os.Stat(m.WorkingDir); err != nil {
		return nil, fmt.Errorf("invalid working directory: '%w'", err)
	}

	m.nextState = make(chan ServerState, 1)
	activity := make(chan pickaxx.Data, 10)

	// start processing state changes
	go eventLoop(m, activity)

	// progress to next state
	m.nextState <- Starting

	return activity, nil
}

// Stop will halt the current process by sending a shutdown command.
// This will kill the process if it does not respond in a given timeframe.
func (m *serverManager) Stop() error {
	log := log.WithField("action", "ProcessManager.Stop()")

	if !m.Running() {
		log.Info("not running")
		return ErrNoProcess
	}

	m.nextState <- Stopping
	return nil
}

// Submit will submit a new command to the underlying Minecraft server.
// Any output is returned asynchonously in the processing loop.
// Prefixed slash-commands will have slashes trimmed (e.g. "/help" -> "help")
func (m *serverManager) Submit(command string) error {
	if m.state != Running && m.state != Stopping {
		return ErrNoProcess
	}

	if len(command) == 0 {
		return errors.New("command is empty")
	}

	if command[0] == '/' {
		command = command[1:]
	}

	cmd := fmt.Sprintf("%s\n", command)
	_, err := io.WriteString(m.cmdIn, cmd)

	return err
}

// Running returns whether the process is running.
func (m *serverManager) Running() bool {
	return m.currentStateIn(Starting, Running)
}

// eventLoop processes state transitions.
func eventLoop(m *serverManager, out chan<- pickaxx.Data) {
	var (
		log = log.WithField("action", "eventLoop")
		wg  = sync.WaitGroup{}

		newState ServerState
		err      error
	)

	mainCtx := context.Background()
	portCheckCtx, stopPortCheck := context.WithCancel(mainCtx)

	defer func() {
		stopPortCheck()
		log.Debug("waiting for child processes to quit")
		wg.Wait() // wait for any child routines to quit

		log.Debug("closing activity channel")
		close(out)
	}()

	for {
		newState = m.setState(<-m.nextState) // blocks until next state transition event

		switch newState {
		case Starting:
			out <- consoleOutput{"Server is starting"}

			if _, err = startServer(mainCtx, m); err != nil {
				log.WithError(err).Error("failed to start server")
			}
		case Running:
			wg.Add(1)
			go func() {
				defer wg.Done()
				pipeOutput(m.cmdOut, out)
			}()

			// start liveness probe
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := checkPort(portCheckCtx, m.Port, time.Second*15, time.Second*2); err != nil {
					out <- consoleOutput{"Process not responding. Initiating shutdown."}
					m.Stop()
				}
			}()
		case Stopping:
			out <- consoleOutput{"Shutting down.."}
			stopPortCheck()
			stopServer(mainCtx, m)
		case Stopped:
			out <- consoleOutput{"Shutdown complete. Thanks for playing."}
		}

		m.notifier.Notify(newState)
		out <- stateChangeEvent{newState}

		if newState == Stopped {
			return
		}
	}
}

// currentStateIn returns true if process is in any of the provided states.
func (m *serverManager) currentStateIn(states ...ServerState) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, s := range states {
		if m.state == s {
			return true
		}
	}
	return false
}

func (m *serverManager) setState(newState ServerState) ServerState {
	m.lock.Lock()
	defer m.lock.Unlock()

	log.WithField("state", fmt.Sprintf("%v->%v", m.state, newState)).Info("state transition")
	m.state = newState

	return newState
}
