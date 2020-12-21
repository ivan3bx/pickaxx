package pickaxx

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/apex/log"
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
)

var (
	// DefaultCommand is the name of the executable.
	DefaultCommand = []string{"java", MaxMem, MinMem, "-jar", JarFile, "nogui"}
)

// ErrProcessExists exists when a new server process can not be started.
var ErrProcessExists = errors.New("unable to start new process")

// ErrNoProcess occurs when no process exists to take an action on.
var ErrNoProcess = errors.New("no process running")

// ErrInvalidClient occurs when a client is not valid.
var ErrInvalidClient = errors.New("client not valid")

// ProcessManager manages the Minecraft server's process lifecycle.
type ProcessManager struct {
	// Command is the command & arguments passed when 'Start()' is
	// invoked. If not set, it will default to 'DefaultExecArgs'.
	Command []string

	cmd     *exec.Cmd
	cmdIn   io.Writer
	cmdOut  io.Writer // output to any/all places
	fileOut *os.File

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

// RecentActivity will return the latest content contained in
// this server's log file.
func (m *ProcessManager) RecentActivity() []string {
	if m.fileOut == nil {
		return []string{}
	}
	content, _ := ioutil.ReadFile(m.fileOut.Name())
	return strings.Split(string(content), "\n")
}

// Start will initialize a new process, sending all output to the provided
// io.Writer, and set values on this object to track process state.
// This will return an error if the process is already running.
func (m *ProcessManager) Start(w io.Writer) error {
	if m.Active() {
		return fmt.Errorf("server already running: %w", ErrProcessExists)
	}

	// initialize
	if len(m.Command) == 0 {
		m.Command = DefaultCommand
	}

	if m.serverPort == 0 {
		m.serverPort = DefaultPort
	}

	m.nextState = make(chan ServerState, 1)

	// set up log file
	logFile, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("pickaxx_%d", m.serverPort))

	if err != nil {
		return err
	}

	m.fileOut = logFile

	// cmdOut collects all output from this process & sends it to both clients & logfile
	m.cmdOut = io.MultiWriter(w, &newlineWriter{logFile})

	go eventLoop(m, w)

	m.nextState <- Starting
	return nil
}

// Stop will halt the current process by sending a direct
// shutdown command. This will also kill the process if it
// does not respond in a given timeframe.
func (m *ProcessManager) Stop() error {
	log := log.WithField("action", "ProcessManager.Stop()")

	if !m.Active() {
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

// Active returns whether the process is running.
func (m *ProcessManager) Active() bool {
	return m.currentStateIn(Starting, Running)
}

// CanAcceptCommands returns true if this instance can direct commands
// through to the process.
func (m *ProcessManager) CanAcceptCommands() bool {
	return m.currentStateIn(Running, Stopping)
}

// eventLoop processes state transitions for a process manager and
// writes a log of activity to the given writer.
func eventLoop(m *ProcessManager, w io.Writer) {
	var (
		log      = log.WithField("action", "eventLoop")
		cmd      *exec.Cmd
		newState ServerState
	)

	for {
		newState = <-m.nextState

		m.lock.Lock()
		log.WithField("state", fmt.Sprintf("%v->%v", m.state, newState)).Info("state transition")
		m.state = newState
		m.lock.Unlock()

		switch newState {
		case Starting:
			cmd, _ = startServer(m)
		case Running:
			stop := m.registerObserver(Stopping, Stopped)
			go m.startLivenessProbe(stop)
		case Stopping:
			stopServer(m, cmd)
		case Stopped:
			io.WriteString(m.cmdOut, "Shutdown complete. Thanks for playing.")
		}

		m.notifier.Notify(newState)

		newState.writeJSON(w)

		if newState == Stopped {
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

func (m *ProcessManager) startLivenessProbe(onStop <-chan ServerState) {
	check := portChecker{
		stop: onStop,
		cancel: func() {
			if m.Active() {
				io.WriteString(m.cmdOut, "Java process not responding. Initiating shutdown.")
				log.WithField("action", "livenessProbe()").Warn("probe failed")
				m.Stop()

			}
		},
	}
	go check.Run("localhost", fmt.Sprintf("%d", m.serverPort))
}
