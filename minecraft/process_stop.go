package minecraft

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/apex/log"
)

func stopServer(ctx context.Context, m *serverManager) {

	var (
		log = log.WithField("action", "ProcessManager.stopServer()")
		cmd = m.cmd
		wg  = sync.WaitGroup{}
	)

	// context used to halt process if clean exit does not complete
	ctx, cancelTimer := context.WithTimeout(ctx, time.Second*10)

	defer func() {
		cancelTimer()          // cancel our timer
		wg.Wait()              // wait for routines to stop
		m.nextState <- Stopped // set terminal state
	}()

	wg.Add(1)

	go func() {
		defer wg.Done()
		waitForTermination(ctx, cmd)
	}()

	log.Info("clean shutdown starting")
	if err := m.Submit("stop"); err != nil {
		log.WithError(err).Warn("unable to send /stop command")
	}

	if _, err := cmd.Process.Wait(); err != nil {
		log.WithError(err).Warn("clean shutdown failed")
	}
}

func waitForTermination(ctx context.Context, cmd *exec.Cmd) {
	<-ctx.Done()

	if ctx.Err() == context.DeadlineExceeded {
		log.Debug("deadline expired. force quit.")
		cmd.Process.Kill()
	}
}
