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

var _ pickaxx.Logger = &logFileTracker{}

type logFileTracker struct {
	consoleLog   *os.File
	clientWriter io.Writer
}

// NewTracker returns a new instance of a tracker that can
// write data out to the provided writers.
func NewTracker(w ...io.Writer) pickaxx.Logger {
	file, err := ioutil.TempFile(os.TempDir(), "pickaxx_log")

	if err != nil {
		log.WithError(err).Error("failed to create temp file")
		return nil
	}

	return &logFileTracker{
		consoleLog:   file,
		clientWriter: io.MultiWriter(w...),
	}
}

func (t *logFileTracker) Write(p []byte) (n int, err error) {
	line := fmt.Sprintf("%s\n", string(p))
	return io.WriteString(t.consoleLog, line)
}

// Track will begin tracking activity from the given channel. This method
// blocks the caller, and will return errors for any unexpected exit, or
// 'nil' when the underlying channel is closed.
func (t *logFileTracker) Track(ch <-chan pickaxx.Data) error {
	log := log.WithField("action", "logFileTracker")
	enc := json.NewEncoder(t.clientWriter)
	// w := &newlineWriter{t.consoleLog}

	for dataItem := range ch {

		if err := enc.Encode(dataItem); err != nil {
			log.WithError(err).Errorf("encoding failure: %v", dataItem)
			return err
		}
		if val, ok := dataItem.(pickaxx.ConsoleData); ok {
			t.Write([]byte(val.String()))
			// // console output gets written to our writer
			// io.WriteString(w, val.String())
		}
	}

	log.Info("stopped")
	return nil
}

// History returns recent entries equal to the number of lines,
// or -1 if all available data should be returned.
func (t *logFileTracker) History(length int) []string {
	content, _ := ioutil.ReadFile(t.consoleLog.Name())
	lines := strings.Split(string(content), "\n")

	if length == -1 || length > len(lines) {
		return lines
	}

	return lines[:length]
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
