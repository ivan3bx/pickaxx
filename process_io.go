package pickaxx

import (
	"bufio"
	"io"
	"os/exec"
	"sync"
)

// newlineWriter is a writer that inserts '\n' newlines after each call.
type newlineWriter struct {
	wrapped io.Writer
}

func (w *newlineWriter) Write(p []byte) (n int, err error) {
	if n, err := w.wrapped.Write(p); err != nil {
		return n, err
	}
	return w.wrapped.Write([]byte("\n"))
}

// pipeCommandOutput creates a goroutine to start receiving
// stdout & stderr from cmd and pipe it to the given destination.
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

	writeOut := make(chan []byte, 5)
	once := sync.Once{}

	writeFunc := func(r io.Reader) {
		s := bufio.NewScanner(r)
		for s.Scan() {
			writeOut <- s.Bytes()
		}
		once.Do(func() { close(writeOut) })
	}

	go writeFunc(cmdOut)
	go writeFunc(cmdErr)

	// funnel output to single destination
	go func() {
		for data := range writeOut {
			dest.Write(data)
		}
	}()

	return nil
}
