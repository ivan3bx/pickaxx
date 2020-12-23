package minecraft

import (
	"os/exec"
	"sync"
	"time"

	"github.com/apex/log"
)

func stopServer(m *ProcessManager, cmd *exec.Cmd) {
	log := log.WithField("action", "ProcessManager.stopServer()")

	cleanExit := make(chan bool, 1)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
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

	wg.Wait() // wait for routines to stop

	m.nextState <- Stopped
}
