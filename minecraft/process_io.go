package minecraft

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/ivan3bx/pickaxx"
)

var (
	_ pickaxx.Data = &consoleOutput{}
	_ pickaxx.Data = &stateChangeOutput{}
)

// consoleOutput represents console output (free-form text data).
type consoleOutput struct {
	Text string
}

func (d consoleOutput) String() string { return d.Text }

// MarshalJSON converts this output to valid JSON.
func (d consoleOutput) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`{"output":"%s"}`, d.Text)
	return []byte(jsonString), nil
}

// stateChangeOutput represents a state transition.
type stateChangeOutput struct {
	State ServerState
}

// MarshalJSON converts this output to valid JSON.
func (d stateChangeOutput) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`{"status":"%s"}`, d.State.String())
	return []byte(jsonString), nil
}

// pipeCommandOutput creates a goroutine to start receiving
// stdout & stderr from cmd and pipe it to the given destination.
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
