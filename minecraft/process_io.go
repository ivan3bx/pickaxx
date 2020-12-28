package minecraft

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"github.com/apex/log"
	"github.com/ivan3bx/pickaxx"
)

var (
	_ pickaxx.Data = &consoleOutput{}
	_ pickaxx.Data = &stateChangeEvent{}
)

// consoleOutput represents console output (free-form text data).
type consoleOutput struct {
	Text string
}

func (d consoleOutput) String() string { return d.Text }

// MarshalJSON converts this output to valid JSON.
func (d consoleOutput) MarshalJSON() ([]byte, error) {
	holder := map[string]string{"output": d.String()}
	return json.Marshal(&holder)
}

// stateChangeEvent represents a state transition event.
type stateChangeEvent struct {
	State ServerState
}

// MarshalJSON converts this output to valid JSON.
func (d stateChangeEvent) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`{"status":"%s"}`, d.State.String())
	return []byte(jsonString), nil
}

// pipeOutput will send all input from the reader as data through the provided channel.
func pipeOutput(r io.Reader, out chan<- pickaxx.Data) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		out <- consoleOutput{s.Text()}
	}

	if err := s.Err(); err != nil {
		log.WithError(err).Error("pipeOutput() closed with error")
		return
	}
}

// pipeCommandOutput returns a reader combining stdout & stderr from the given command.
func pipeCommandOutput(cmd *exec.Cmd) (io.Reader, error) {
	var (
		cmdOut, cmdErr io.Reader
		err            error
	)

	if cmdOut, err = cmd.StdoutPipe(); err != nil {
		return nil, err
	}

	if cmdErr, err = cmd.StderrPipe(); err != nil {
		return nil, err
	}

	return io.MultiReader(cmdOut, cmdErr), nil
}
