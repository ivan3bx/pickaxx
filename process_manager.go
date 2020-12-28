package pickaxx

import (
	"encoding/json"
	"errors"
	"io"
)

// ErrProcessExists exists when a new server process can not be started.
var ErrProcessExists = errors.New("unable to start new process")

// Data is anything that is emitted by a process manager.
type Data json.Marshaler

// ConsoleData is Data that can appear in console output.
type ConsoleData interface {
	Data
	String() string
}

// ProcessManager can manage and interact with a process.
type ProcessManager interface {

	// Start will connect & initiate the underlying process.
	Start() (<-chan Data, error)

	// Stop will halt the process and release any resources.
	Stop() error

	// Running returns true if the underlying process is active, false otherwise.
	Running() bool

	// Submit will send a command to the underlying process.
	Submit(command string) error
}

// Logger emits activity for a stream of Data.
type Logger interface {
	io.Writer

	// Track will begin tracking activity from the given channel. This method
	// blocks until the provided context is canceled, any unexpected error, or
	// if the underlying channel is closed.
	Track(<-chan Data) error

	// History returns recent entries equal to the number of lines,
	// or -1 if all available data should be returned.
	History(lines int) []string
}
