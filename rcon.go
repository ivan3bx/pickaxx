package pickaxx

import (
	"time"

	rcon "github.com/Kelwing/mc-rcon"
	"github.com/apex/log"
)

type consoleInput struct {
	rcon.MCConn
	stop chan<- bool
}

// configureInput creates a new MCConn. It accepts a 'stop' channel which
// it uses to signal an abnormal error condition, and a completion
// function that will be invoked once the console validates the server
// is responding to commands.
//
// This function will start a go routine that routinely checks for
// liveness of the connection by executing a dummy command. If this
// probe fails, the 'stop' channel will receive a message as well.
func configureInput(stop chan<- bool) *consoleInput {
	conn := consoleInput{stop: stop}
	return &conn
}

func (c *consoleInput) SendCommand(command string) (string, error) {
	defer func() {
		if recover() != nil {
			log.Error("error sending command. ignoring")
		}
	}()

	return c.MCConn.SendCommand(command)
}

func (c *consoleInput) connect(host, password string, onConnect func()) {
	var err error

	go func() {
		defer func() {
			c.stop <- true
		}()

		if err := c.open(host, password); err != nil {
			log.WithError(err).Error("rcon: server not detected. failed.")
			return
		}

		if err = c.Authenticate(); err != nil {
			log.WithError(err).Error("rcon: auth failed")
			return
		}

		onConnect()

		livenessProbe(c, time.Second*4)
	}()
}

func (c *consoleInput) open(host, password string) error {
	var err error

	ticker := time.NewTicker(time.Second * 3)
	for i := 0; i < 10; i++ {
		<-ticker.C
		if err = c.Open("localhost:25575", "passw"); err == nil {
			break
		}
	}

	return err
}

func livenessProbe(c *consoleInput, d time.Duration) {
	tick := time.NewTicker(d)
	for range tick.C {
		if _, err := c.SendCommand("list"); err != nil {
			log.Warn("rcon: console is closed")
			return
		}
	}
}
