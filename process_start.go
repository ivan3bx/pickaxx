package pickaxx

import (
	"context"
	"io"
	"os/exec"

	"github.com/apex/log"
)

func startServer(m *ProcessManager) (*exec.Cmd, error) {
	ctx := log.NewContext(context.Background(), log.WithField("action", "ProcessManager.startServer()"))
	ctx, cancel := context.WithCancel(ctx)

	log := log.FromContext(ctx)

	cmd := exec.CommandContext(ctx, m.Command[0], m.Command[1:]...)
	cmd.Dir = m.WorkingDir

	io.WriteString(m.cmdOut, "Server is starting")

	defer func() {
		switch {
		case cmd.Process == nil:
			m.nextState <- Stopping
			cancel()
		default:
			m.nextState <- Running
		}
	}()

	if err := pipeCommandOutput(cmd, m.cmdOut); err != nil {
		log.WithError(err).Error("unable to pipe output")
		return cmd, err
	}

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
