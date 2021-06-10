package main

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/ivan3bx/pickaxx"
)

type loggingReporter struct {
	logFile *os.File
	writer  pickaxx.Writer
}

func (r *loggingReporter) Report(ch <-chan pickaxx.Data) {
	file, err := ioutil.TempFile(os.TempDir(), "pickaxx_log")

	if err != nil {
		log.WithError(err).Error("failed to create temp file")
	}

	r.logFile = file
	defer func() {
		r.logFile.Close()
		log.Debug("reporter closed")
	}()

	for data := range ch {
		r.writeConsoleOutput(data)
		r.writer.Write(data)
	}
}

func (r *loggingReporter) writeConsoleOutput(data pickaxx.Data) {
	if consoleData, ok := data.(pickaxx.ConsoleData); ok {
		io.WriteString(r.logFile, consoleData.String())
		r.logFile.Write([]byte{'\n'})
	}
}

func (r *loggingReporter) ConsoleOutput() []string {
	content, _ := ioutil.ReadFile(r.logFile.Name())
	return strings.Split(string(content), "\n")
}
