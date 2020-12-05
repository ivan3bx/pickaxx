package pickaxx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
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
	cmdIn   io.Writer
	fileOut *os.File

	serverPort int
	state      ServerState
	stop       chan bool
}

// CurrentState is the current running state of the process being managed.
func (m *ProcessManager) CurrentState() ServerState {
	return m.state
}

// Running returns whether the process is running / active.
func (m *ProcessManager) Running() bool {
	return m.state == Starting || m.state == Running
}

func (m *ProcessManager) Logfile() string {
	return m.fileOut.Name()
}

// Start will initialize a new process, sending all output to the provided
// io.Writer, and set values on this object to track process state.
// This will return an error if the process is already running.
func (m *ProcessManager) Start(w io.Writer) error {
	if m.state == Running || m.state == Starting {
		return fmt.Errorf("server already running: %w", ErrProcessExists)
	}

	// initialize
	m.serverPort = 25565
	m.state = Unknown
	m.stop = make(chan bool, 1)
	m.writeState(w, Starting)

	// set up log file
	logFile, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("pickaxx_%d", m.serverPort))

	if err != nil {
		return err
	}

	m.fileOut = logFile

	// cmdOut collects all output from this process
	consoleOut := io.MultiWriter(w, &newlineWriter{logFile})

	go func() {

		io.WriteString(consoleOut, "Server is starting")

		ctx := log.NewContext(context.Background(), log.WithField("action", "processManager"))
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		log := log.FromContext(ctx)
		cmd := exec.CommandContext(ctx, "java", MaxMem, MinMem, "-jar", JarFile, "nogui")
		cmd.Dir = "testserver"

		if err := pipeCommandOutput(cmd, consoleOut); err != nil {
			log.WithError(err).Error("error piping command output")
			return
		}

		pipin, err := cmd.StdinPipe()

		if err != nil {
			log.WithError(err).Error("error connecting command stdinput")
			return
		}

		m.cmdIn = pipin

		if err := cmd.Start(); err != nil {
			log.WithError(err).Error("failed to start command")
			m.writeState(w, Stopped)
			return
		}

		m.writeState(w, Running)

		// liveness probe
		go livenessProbe(func() { m.Stop() })

		//
		// wait until server is stopped
		//

		<-m.stop
		m.writeState(w, Stopping)
		io.WriteString(consoleOut, "Shutting down..")

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
		m.Submit("stop")
		time.Sleep(time.Second * 5)

		if _, err := cmd.Process.Wait(); err != nil {
			log.WithError(err).Warn("clean shutdown failed with error")
		}

		m.writeState(w, Stopped)
		io.WriteString(consoleOut, "Shutdown complete. Thanks for playing.")
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

// Submit will submit a new command to the underlying minecraft process.
// Any output is returned asynchonously in the processing loop.
// Prefixed slash-commands will have slashes trimmed (e.g. "/help" -> "help")
func (m *ProcessManager) Submit(command string) error {
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
		dest.Write(s.Bytes())
	}()

	return nil
}

func (m *ProcessManager) writeState(w io.Writer, newState ServerState) error {
	m.state = newState
	jsonString := fmt.Sprintf(`{"status":"%s"}`, newState.String())
	_, err := w.Write([]byte(jsonString))
	return err
}

func livenessProbe(cancel func()) {
	time.Sleep(time.Second * 15)
	ticker := time.NewTicker(time.Second * 2)
	defer cancel()

	for {
		<-ticker.C
		if !portOpen("localhost", "25565") {
			return
		}
	}
}

func portOpen(host string, port string) bool {
	hostname := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", hostname, time.Second)

	if err != nil || conn == nil {
		return false
	}

	conn.Close()
	return true
}

type newlineWriter struct {
	wrapped io.Writer
}

func (w *newlineWriter) Write(p []byte) (n int, err error) {
	if n, err := w.wrapped.Write(p); err != nil {
		return n, err
	}
	return w.wrapped.Write([]byte("\n"))
}
