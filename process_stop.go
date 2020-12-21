package pickaxx

import (
	"io"
	"os/exec"
	"time"

	"github.com/apex/log"
)

func stopServer(m *ProcessManager, cmd *exec.Cmd) {
	log := log.WithField("action", "ProcessManager.stopServer()")

	io.WriteString(m.cmdOut, "Shutting down..")

	cleanExit := make(chan bool, 1)

	go func() {
		timer := time.NewTimer(time.Second * 10)

		select {
		case <-cleanExit:
			return
		case <-timer.C:
			log.Debug("deadline expired. force quit.")
			if err := cmd.Process.Kill(); err != nil {
				log.WithError(err).Debug("kill failed")
			}
		}
	}()

	log.Info("clean shutdown starting")
	if err := m.Submit("stop"); err != nil {
		log.WithError(err).Warn("unable to send /stop command")
		cmd.Process.Kill()
	}

	if _, err := cmd.Process.Wait(); err != nil {
		log.WithError(err).Warn("clean shutdown failed")
		return
	}

	cleanExit <- true
	m.nextState <- Stopped
}
