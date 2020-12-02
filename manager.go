package pickaxx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	rcon "github.com/Kelwing/mc-rcon"

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

// ProcessManager manages the Minecraft server's process lifecycle.
type ProcessManager struct {
	state   ServerState
	process *os.Process

	console        *rcon.MCConn
	writer         io.Writer
	killServerFunc context.CancelFunc
}

// Running returns whether the process is running / active.
func (m *ProcessManager) Running() bool {
	return m.state == Starting || m.state == Running
}

// Start will initialize a new process, sending all output to the provided
// io.Writer, and set values on this object to track process state.
// This will return an error if the process is already running.
func (m *ProcessManager) Start(w io.Writer) error {
	if m.state == Running {
		return fmt.Errorf("server already running: %w", ErrProcessExists)
	}

	// initialize
	m.state = Unknown
	m.writer = w

	m.updateState(Starting)

	ctx := log.NewContext(context.Background(), log.WithField("action", "startServer"))
	ctx, cancel := context.WithCancel(ctx)
	log := log.FromContext(ctx)

	cmd := exec.CommandContext(ctx, "java", MaxMem, MinMem, "-jar", JarFile, "nogui")
	cmd.Dir = "testserver"

	m.killServerFunc = cancel

	var (
		cmdOut io.Reader
		cmdErr io.Reader

		err error
	)

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

	go m.captureOutput(cmdOut)
	go m.captureOutput(cmdErr)
	go m.configureInput(ctx)

	return nil
}

// func captureOutput(cmd *exec.Cmd, w io.Writer) err{

// }

// Stop will halt the current process by sending a direct
// shutdown command. This will also kill the process if it
// does not respond in a given timeframe.
func (m *ProcessManager) Stop(ctx context.Context) error {
	if !m.Running() {
		m.updateState(Stopped)
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
	m.console.SendCommand("stop")

	if _, err := m.process.Wait(); err != nil {
		log.WithError(err).Warn("clean shutdown failed with error")
	}

	m.process = nil
	m.updateState(Stopped)
	log.Info("clean shutdown completed")

	return nil
}

func (m *ProcessManager) captureOutput(src io.Reader) {
	s := bufio.NewScanner(src)
	w := m.writer

	for s.Scan() {
		w.Write(s.Bytes())
	}
}

func (m *ProcessManager) updateState(newState ServerState) {
	m.state = newState

	if m.writer == nil {
		return
	}

	jsonString := fmt.Sprintf(`{"status":"%s"}`, newState.String())
	m.writer.Write([]byte(jsonString))
}
