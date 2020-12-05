package pickaxx

import (
	"bytes"
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
	initialized bool
	output      chan map[string]interface{}
	pool        map[string]*websocketClient
}

func (c *ClientManager) initialize() {
	c.Lock()
	defer c.Unlock()

	if c.initialized {
		return
	}

	c.pool = map[string]*websocketClient{}
	c.output = make(chan map[string]interface{}, 1)

	go func() {
		for {
			if val := <-c.output; val != nil {
				c.broadcast(val)
				continue
			}
		}
	}()

	c.initialized = true
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
	holder := map[string]interface{}{}

	// Send well-formed JSON as-is or wrap it as generic 'output'
	if err := json.Unmarshal(data, &holder); err != nil {
		holder = map[string]interface{}{"output": string(data)}
	}

	c.output <- holder
	return len(data), nil
}

func (c *ClientManager) broadcast(data map[string]interface{}) error {
	buf := bytes.Buffer{}

	if err := json.NewEncoder(&buf).Encode(&data); err != nil {
		return err
	}

	for remoteAddr, client := range c.pool {
		err := client.WriteMessage(websocket.TextMessage, buf.Bytes())

		if err != nil {
			log.WithField("remoteAddr", remoteAddr).Warn("client disconnected")
			delete(c.pool, remoteAddr)
			return err
		}
	}

	return nil
}
