package pickaxx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
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

// ProcessManager manages the Minecraft server's process lifecycle.
type ProcessManager struct {
	state ServerState
	stop  chan bool

	console *consoleInput
}

// CurrentState is the current running state of the process being managed.
func (m *ProcessManager) CurrentState() ServerState {
	return m.state
}

// Running returns whether the process is running / active.
func (m *ProcessManager) Running() bool {
	return m.state == Starting || m.state == Running
}

// Start will initialize a new process, sending all output to the provided
// io.Writer, and set values on this object to track process state.
// This will return an error if the process is already running.
func (m *ProcessManager) Start(w io.Writer) error {
	if m.state == Running || m.state == Starting {
		return fmt.Errorf("server already running: %w", ErrProcessExists)
	}
	fmt.Println("State is:", m.state.String())

	// initialize
	m.state = Unknown
	m.stop = make(chan bool, 1)

	m.writeState(w, Starting)

	go func() {
		ctx := log.NewContext(context.Background(), log.WithField("action", "startServer"))
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		log := log.FromContext(ctx)
		cmd := exec.CommandContext(ctx, "java", MaxMem, MinMem, "-jar", JarFile, "nogui")
		cmd.Dir = "testserver"

		pipeCommandOutput(cmd, w)

		if err := cmd.Start(); err != nil {
			log.WithError(err).Error("failed to start command")
			m.writeState(w, Stopped)
			return
		}

		m.console = &consoleInput{stop: m.stop}
		m.console.connect("localhost:25575", "passw", func() {
			m.writeState(w, Running)
		})

		//
		// wait until server is stopped
		//

		<-m.stop
		m.writeState(w, Stopping)

		go func() {
			timer := time.NewTimer(time.Second * 10)

			select {
			case <-ctx.Done():
			case <-timer.C:
				log.Debug("timer deadline expired")
				cancel()
			}

		}()

		log.Info("clean shutdown starting..")
		m.console.SendCommand("stop")

		if _, err := cmd.Process.Wait(); err != nil {
			log.WithError(err).Warn("clean shutdown failed with error")
		}

		m.writeState(w, Stopped)
		log.Info("shutdown completed")
	}()

	return nil
}

// Stop will halt the current process by sending a direct
// shutdown command. This will also kill the process if it
// does not respond in a given timeframe.
func (m *ProcessManager) Stop() error {
	if !m.Running() {
		return ErrNoProcess
	}

	m.stop <- true
	return nil
}

func pipeCommandOutput(cmd *exec.Cmd, dest io.Writer) error {
	var (
		cmdOut, cmdErr io.Reader
		err            error
	)

	if cmdOut, err = cmd.StdoutPipe(); err != nil {
		return err
	}

	if cmdErr, err = cmd.StderrPipe(); err != nil {
		return err
	}

	go func() {
		s := bufio.NewScanner(cmdOut)
		for s.Scan() {
			dest.Write(s.Bytes())
		}
	}()

	go func() {
		s := bufio.NewScanner(cmdErr)
		for s.Scan() {
			dest.Write(s.Bytes())
		}
	}()

	return nil
}

func (m *ProcessManager) writeState(w io.Writer, newState ServerState) error {
	m.state = newState
	jsonString := fmt.Sprintf(`{"status":"%s"}`, newState.String())
	_, err := w.Write([]byte(jsonString))
	return err
}
