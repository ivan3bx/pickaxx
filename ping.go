package pickaxx

import (
	"time"

	"github.com/apex/log"
	"github.com/gorilla/websocket"
)

const frequency = time.Second * 10
const rspTimeout = time.Second * 5

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
