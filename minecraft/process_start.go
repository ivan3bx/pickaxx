package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"os"
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/ivan3bx/pickaxx"
)

var _ pickaxx.Monitor = &LogfileMonitor{}

// LogfileMonitor monitors console output and sends data to a log file.
type LogfileMonitor struct {
	logFile *os.File
}

// Accept takes a channel and starts writing ConsoleData entries to a log file.
func (m *LogfileMonitor) Accept(ch <-chan pickaxx.Data) error {
	log := log.WithField("action", "LogfileMonitor.Accept()")

	file, err := ioutil.TempFile(os.TempDir(), "pickaxx_log")

	if err != nil {
		log.WithError(err).Error("failed to create temp file")
		return nil
	}

	m.logFile = file

	for data := range ch {
		if val, ok := data.(pickaxx.ConsoleData); ok {
			line := fmt.Sprintf("%s\n", val.String())
			io.WriteString(file, line)
		}
	}

	log.Debug("channel closed")
	return nil
}

// History returns recent entries equal to the number of lines in the param.
// If length=-1, all available data is returned.
func (m *LogfileMonitor) History(length int) []string {
	content, _ := ioutil.ReadFile(m.logFile.Name())
	lines := strings.Split(string(content), "\n")

	if length == -1 || length > len(lines) {
		return lines
	}

	return lines[:length]
}

// PassThruMonitor monitors activity and serializes it as JSON through a provided writer.
type PassThruMonitor struct {
	Writer io.Writer
}

// Accept takes a channel and starts writing ConsoleData entries to a log file.
func (m *PassThruMonitor) Accept(ch <-chan pickaxx.Data) error {
	log := log.WithField("action", "PassThruMonitor.Accept()")
	enc := json.NewEncoder(m.Writer)

	for data := range ch {
		enc.Encode(data)
	}
	log.Debug("channel closed")
	return nil
}

func startServer(ctx context.Context, m *serverManager) (*exec.Cmd, error) {
	ctx, cancel := context.WithCancel(ctx)

	log := log.WithField("action", "ProcessManager.startServer()")

	cmd := exec.CommandContext(ctx, m.Command[0], m.Command[1:]...)
	cmd.Dir = m.WorkingDir

	defer func() {
		switch {
		case cmd.Process == nil:
			cancel()
			m.nextState <- Stopping
		default:
			m.nextState <- Running
		}
	}()

	pipout, err := pipeCommandOutput(cmd)

	if err != nil {
		log.WithError(err).Error("unable to pipe output")
		return cmd, err
	}

	m.cmdOut = pipout

	pipin, err := cmd.StdinPipe()

	if err != nil {
		log.WithError(err).Error("unable to pipe input")
		return cmd, err
	}

	m.cmdIn = pipin

	if err := cmd.Start(); err != nil {
		log.WithError(err).Error("command failed")
		return cmd, err
	}

	m.cmd = cmd

	return cmd, nil
}
