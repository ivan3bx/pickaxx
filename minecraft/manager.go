package minecraft

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/apex/log"
	"github.com/ivan3bx/pickaxx"
)

var _ pickaxx.ProcessManager = &ProcessManager{}

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

var (
	// DefaultCommand is the name of the executable.
	DefaultCommand = []string{"java", MaxMem, MinMem, "-jar", JarFile, "nogui"}
)

// ErrNoProcess occurs when no process exists to take an action on.
var ErrNoProcess = errors.New("no process running")

// ProcessManager manages the Minecraft server's process lifecycle.
type ProcessManager struct {
	// Command is the command & arguments passed when 'Start()' is
	// invoked. If not set, it will default to 'DefaultExecArgs'.
	Command []string

	// WorkingDir is our starting directory. If not set, will default
	// to 'DefaultWorkignDir'
	WorkingDir string

	cmd    *exec.Cmd
	cmdIn  io.Writer
	cmdOut io.Reader

	serverPort int

	// state transition
	state     ServerState
	lock      sync.RWMutex
	nextState chan ServerState

	// observers of state transitions
	notifier StatusNotifier
}

func (m *ProcessManager) registerObserver(states ...ServerState) <-chan ServerState {
	return m.notifier.Register(states)
}

func (m *ProcessManager) unregister(ch <-chan ServerState) {
	m.notifier.Unregister(ch)
}

// Start will initialize a new process, sending all output to the provided
// channel, and set values on this object to track process state.
// This will return an error if the process is already running.
func (m *ProcessManager) Start() (<-chan pickaxx.Data, error) {
	if m.Running() {
		return nil, fmt.Errorf("server already running: %w", pickaxx.ErrProcessExists)
	}

	// initialize
	if len(m.Command) == 0 {
		m.Command = DefaultCommand
	}

	if m.serverPort == 0 {
		m.serverPort = DefaultPort
	}

	if m.WorkingDir == "" {
		m.WorkingDir = DefaultWorkingDir
	}

	activityCh := make(chan pickaxx.Data, 10)
	stateCh := make(chan ServerState, 1)

	m.nextState = stateCh

	if _, err := os.Stat(m.WorkingDir); err != nil {
		return nil, fmt.Errorf("invalid working directory: '%w'", err)
	}

	// start processing state changes
	go eventLoop(m, activityCh)

	// progress to next state
	m.nextState <- Starting

	return activityCh, nil
}

// Stop will halt the current process by sending a direct
// shutdown command. This will also kill the process if it
// does not respond in a given timeframe.
func (m *ProcessManager) Stop() error {
	log := log.WithField("action", "ProcessManager.Stop()")

	if !m.Running() {
		log.Info("not running")
		return ErrNoProcess
	}

	m.nextState <- Stopping
	return nil
}

// Submit will submit a new command to the underlying minecraft process.
// Any output is returned asynchonously in the processing loop.
// Prefixed slash-commands will have slashes trimmed (e.g. "/help" -> "help")
func (m *ProcessManager) Submit(command string) error {
	if !m.CanAcceptCommands() {
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

// CurrentState is the current running state of the process being managed.
func (m *ProcessManager) CurrentState() ServerState {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.state
}

// Running returns whether the process is running.
func (m *ProcessManager) Running() bool {
	return m.currentStateIn(Starting, Running)
}

// CanAcceptCommands returns true if this instance can direct commands
// through to the process.
func (m *ProcessManager) CanAcceptCommands() bool {
	return m.currentStateIn(Running, Stopping)
}

// eventLoop processes state transitions for a process manager and
// writes a log of activity to the given writer.
func eventLoop(m *ProcessManager, out chan<- pickaxx.Data) {
	var (
		log      = log.WithField("action", "eventLoop")
		cmd      *exec.Cmd
		newState ServerState
	)

	wg := sync.WaitGroup{}

	for {
		newState = <-m.nextState

		m.lock.Lock()
		log.WithField("state", fmt.Sprintf("%v->%v", m.state, newState)).Info("state transition")
		m.state = newState
		m.lock.Unlock()

		switch newState {
		case Starting:
			out <- consoleOutput{"Server is starting"}
			cmd, _ = startServer(m)
		case Running:
			stop := m.registerObserver(Stopping, Stopped)

			// capture any output
			wg.Add(1)
			go func() {
				defer wg.Done()
				s := bufio.NewScanner(m.cmdOut)
				for s.Scan() {
					out <- consoleOutput{s.Text()}
				}
			}()

			// start liveness probe
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := m.runProbe(stop); err != nil {
					out <- consoleOutput{"Java process not responding. Initiating shutdown."}
					log.WithField("action", "livenessProbe()").Warn("probe failed")
					m.Stop()
				}
			}()

		case Stopping:
			out <- consoleOutput{"Shutting down.."}
			stopServer(m, cmd)
		case Stopped:
			out <- consoleOutput{"Shutdown complete. Thanks for playing."}
		}

		m.notifier.Notify(newState)

		out <- stateChangeOutput{newState}

		if newState == Stopped {
			log.Debug("waiting for child processes to quit")
			wg.Wait() // wait for any child routines to quit
			log.Debug("event loop closing")
			close(out)
			return
		}
	}
}

// currentStateIn returns true if the current process is in any
// of the provided states.
func (m *ProcessManager) currentStateIn(states ...ServerState) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, s := range states {
		if m.state == s {
			return true
		}
	}
	return false
}

func (m *ProcessManager) runProbe(onStop <-chan ServerState) error {
	check := portChecker{stop: onStop}
	return check.Run("localhost", fmt.Sprintf("%d", m.serverPort))
}
