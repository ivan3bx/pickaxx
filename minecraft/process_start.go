package minecraft

import (
	"context"
	"os/exec"

	"github.com/apex/log"
)

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
