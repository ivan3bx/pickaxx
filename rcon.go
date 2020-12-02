package pickaxx

import (
	"context"
	"fmt"
	"time"

	rcon "github.com/Kelwing/mc-rcon"
	"github.com/apex/log"
)

func (m *ProcessManager) configureInput(ctx context.Context) {
	var (
		conn *rcon.MCConn
		err  error
	)

	conn = &rcon.MCConn{}

	ticker := time.NewTicker(time.Second * 3)

	for i := 0; i < 10; i++ {
		<-ticker.C
		if err = conn.Open("localhost:25575", "passw"); err != nil {
			log.Debug("rcon: server not responding yet")
			continue
		}
		break
	}

	if err != nil {
		log.Error("rcon: server not detected. failed.")
		m.Stop(ctx)
		return
	}

	if err = conn.Authenticate(); err != nil {
		log.WithError(err).Error("rcon: auth failed")
		m.Stop(ctx)
		return
	}

	// server is running!
	m.console = conn
	m.updateState(Running)

	// detect server exiting
	m.consoleLoop(ctx)
}

func (m *ProcessManager) consoleLoop(ctx context.Context) {
	conn := m.console

	if conn == nil {
		return
	}

	tick := time.NewTicker(time.Second * 2)

	for range tick.C {
		_, err := conn.SendCommand("list")

		if err != nil {
			log.Error("process not responding")
			m.Stop(ctx)
			return
		}

		fmt.Println("tick..")
	}
}
