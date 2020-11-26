package pickaxx

import "github.com/gorilla/websocket"

type client struct {
	manager *serverManager
	conn    *websocket.Conn
}

type clients map[*client]bool

func (c clients) Add(newClient *client) {
	c[newClient] = true
}

func (c clients) Remove(client *client) {
	if _, ok := c[client]; ok {
		delete(c, client)
	}
}

func (c clients) broadcast(data map[string]interface{}) {
	for client := range c {
		client.conn.WriteJSON(data)
	}
}
