package pickaxx

import "github.com/gorilla/websocket"

// Client describes a user currently interacting with the system.
type Client interface {
}

type client struct {
	manager *serverManager
	conn    *websocket.Conn
}

// ClientPool is a collection of clients
type ClientPool map[*client]bool

//Add will add a client to the pool. If the Client already exists, no change occurs.
func (c ClientPool) Add(newClient *client) {
	c[newClient] = true
}

//Remove will remove a client to the pool. If the Client does not exist, no change occurs.
func (c ClientPool) Remove(client *client) {
	if _, ok := c[client]; ok {
		delete(c, client)
	}
}

func (c ClientPool) broadcast(data map[string]interface{}) {
	for client := range c {
		client.conn.WriteJSON(data)
	}
}
