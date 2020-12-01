package pickaxx

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/apex/log"
	"github.com/gorilla/websocket"
)

type websocketClient struct {
	*websocket.Conn
}

var _ io.Writer = &ClientManager{}

// ClientManager is a collection of clients
type ClientManager struct {
	sync.Mutex
	output chan []byte
	pool   map[string]*websocketClient
}

func (c *ClientManager) initialize() {
	c.Lock()
	defer c.Unlock()

	if c.pool == nil {
		c.pool = map[string]*websocketClient{}
	}

	if c.output == nil {
		// start run loop for broadcasting to clients
		c.output = make(chan []byte, 1)

		go func() {
			for {
				select {
				case data := <-c.output:
					c.broadcast(data)
				}
			}
		}()
	}

}

// AddClient adds a new client to this manager
func (c *ClientManager) AddClient(conn *websocket.Conn) {
	c.initialize()
	client := websocketClient{conn}
	c.pool[conn.RemoteAddr().String()] = &client
}

// Write will send data down a channel to be sent to clients. This
// operation must write to a channel, as writes to an underlying
// websocket can not happen concurrently.
func (c *ClientManager) Write(data []byte) (int, error) {
	c.output <- data
	return len(data), nil
}

func (c *ClientManager) broadcast(data interface{}) error {
	if byteData, ok := data.([]byte); ok {
		holder := map[string]string{}

		// Send well-formed JSON as-is or wrap it as generic 'output'
		if err := json.Unmarshal(byteData, &holder); err != nil {
			holder = map[string]string{"output": string(byteData)}
		}

		data = holder
	}

	for remoteAddr, client := range c.pool {
		if err := client.WriteJSON(data); err != nil {
			log.WithField("remoteAddr", remoteAddr).Warn("client disconnected")
			delete(c.pool, remoteAddr)
			return err
		}
	}

	return nil
}
