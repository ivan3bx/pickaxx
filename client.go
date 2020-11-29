package pickaxx

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

type websocketClient struct {
	*websocket.Conn
}

var _ io.Writer = &ClientManager{}

// ClientManager is a collection of clients
type ClientManager struct {
	sync.Mutex
	pool map[*websocketClient]bool
}

// AddClient adds a new client to this manager
func (c *ClientManager) AddClient(conn *websocket.Conn) {
	c.Lock()
	if c.pool == nil {
		c.pool = map[*websocketClient]bool{}
	}
	c.Unlock()

	client := websocketClient{conn}
	c.pool[&client] = true
}

func (c *ClientManager) Write(data []byte) (int, error) {
	var (
		holder map[string]string
	)

	// Send well-formed JSON as-is or wrap it as generic 'output'
	if err := json.Unmarshal(data, &holder); err != nil {
		holder = map[string]string{"output": string(data)}
	}

	return len(data), c.broadcast(holder)
}

func (c *ClientManager) broadcast(data interface{}) error {
	for client := range c.pool {
		if err := client.WriteJSON(data); err != nil {
			return err
		}
	}
	return nil
}
