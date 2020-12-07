package pickaxx

import (
	"time"

	"github.com/apex/log"
	"github.com/gorilla/websocket"
)

const frequency = time.Second * 10
const rspTimeout = time.Second * 5

// pingLoop will recurringly ping clients. Any client that
// fails to respond will be removed from the client pool.
func pingLoop(c *ClientManager, done chan bool) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for addr, client := range c.pool {
				if err := ping(client.Conn); err != nil {
					delete(c.pool, addr)
				}
			}
		case <-done:
			done <- true // propogate

			for _, client := range c.pool {
				client.Close()
			}

			return
		}
	}
}

func ping(conn *websocket.Conn) error {
	var (
		deadline = time.Now().Add(rspTimeout)
		msg      = websocket.PingMessage
		data     = []byte("nerb")
		err      error
	)

	if err = conn.WriteControl(msg, data, deadline); err != nil {
		log.WithField("host", conn.RemoteAddr()).Warn("client failed ping")
		conn.Close()
	}

	return err
}
