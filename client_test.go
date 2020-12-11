package pickaxx_test

import (
	"testing"

	"github.com/gorilla/websocket"
	"github.com/ivan3bx/pickaxx"
	"github.com/stretchr/testify/assert"
)

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
				m := pickaxx.ClientManager{}
				defer m.Close()

				m.AddClient(<-server.ConnectedSockets)
				assert.NoError(t, client.WaitReceive(websocket.PingMessage, ""))
			},
		},
		{
			name: "server wraps plain text as JSON to client",
			checkFunc: func(t *testing.T, client *TestClient) {
				m := pickaxx.ClientManager{}
				defer m.Close()

				m.AddClient(<-server.ConnectedSockets)
				m.Write([]byte("hi there"))

				assert.NoError(t, client.WaitReceive(websocket.TextMessage, `{"output":"hi there"}`))
			},
		},
		{
			name: "server serializes JSON directly to client",
			checkFunc: func(t *testing.T, client *TestClient) {
				m := pickaxx.ClientManager{}
				defer m.Close()

				m.AddClient(<-server.ConnectedSockets)
				m.Write([]byte(`{"status":"Running"}`))

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
