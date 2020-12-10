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

func (wc *websocketClient) Write(data []byte) error {
	var (
		conn = wc.Conn
		err  error
	)
	if err = conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.WithField("host", wc.RemoteAddr()).Warn("failed to write to client")
		conn.Close()
	}

	return err
}

var _ io.Writer = &ClientManager{}

// ClientManager is a collection of clients
type ClientManager struct {
	sync.Mutex
	initialized bool
	done        chan bool
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
	c.done = make(chan bool, 1)

	go outputLoop(c, c.done)
	go pingLoop(c, c.done)

	c.initialized = true
}

// Close will close any client connections and clean up resources used by this manager.
func (c *ClientManager) Close() error {
	c.done <- true
	return nil
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

	// Send well-formed JSON as-is; wrap anything else as 'output'
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

	for addr, client := range c.pool {
		if err := client.Write(buf.Bytes()); err != nil {
			delete(c.pool, addr)
		}
	}

	return nil
}

func outputLoop(c *ClientManager, done chan bool) {
	for {
		select {
		case val := <-c.output:
			c.broadcast(val)
		case <-done:
			done <- true // propogate
			return
		}
	}
}
