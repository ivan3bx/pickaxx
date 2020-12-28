package pickaxx

import (
	"encoding/json"
	"testing"

	"github.com/apex/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetLevel(log.ErrorLevel)
}

func TestClientManager(t *testing.T) {

	server := MockServer{}

	server.Start(t)
	defer server.Close()

	tests := []struct {
		name      string
		checkFunc func(*testing.T, *TestClient)
	}{
		{
			name: "adding client starts pinging",
			checkFunc: func(t *testing.T, client *TestClient) {
				m := ClientManager{}
				defer m.Close()

				m.AddClient(<-server.ConnectedSockets)
				assert.NoError(t, client.WaitReceive(websocket.PingMessage, ""))
			},
		},
		{
			name: "removes client if pings fail",
			checkFunc: func(t *testing.T, client *TestClient) {
				m := ClientManager{}
				defer m.Close()

				serverConn := <-server.ConnectedSockets // fetch new connection
				serverConn.Close()                      // close it
				m.AddClient(serverConn)                 // add client (writes should fail)

				assert.Error(t, client.WaitReceive(websocket.PingMessage, ""))
			},
		},
		{
			name: "server wraps plain text as JSON to client",
			checkFunc: func(t *testing.T, client *TestClient) {
				m := ClientManager{}
				defer m.Close()

				m.AddClient(<-server.ConnectedSockets)
				m.Write(dummyEvent{"output", "hi there"})

				assert.NoError(t, client.WaitReceive(websocket.TextMessage, `{"output":"hi there"}`))
			},
		},
		{
			name: "server serializes JSON directly to client",
			checkFunc: func(t *testing.T, client *TestClient) {
				m := ClientManager{}
				defer m.Close()

				m.AddClient(<-server.ConnectedSockets)
				m.Write(dummyEvent{"status", "Running"})

				assert.NoError(t, client.WaitReceive(websocket.TextMessage, `{"status":"Running"}`))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			// connect our client to mock server
			client := TestClient{}
			client.Connect(server.WebsocketURL)
			defer client.Close()

			// call our test
			tc.checkFunc(t, &client)
		})
	}
}

type dummyEvent struct {
	key, value string
}

func (e dummyEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{e.key: e.value})
}
