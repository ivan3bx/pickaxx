package pickaxx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gorilla/websocket"
)

type WebSocketData struct {
	messageType int
	data        []byte
}

// MockServer is an HTTP Server whose only job is to
// receive Websocket upgrade requests.
type MockServer struct {
	WebsocketURL     string
	srv              *httptest.Server
	ConnectedSockets chan *websocket.Conn
}

// Start will initialize the web server. It accepts WebSocket upgrade requests
// and will send these connections through a channel which tests can read from.
func (m *MockServer) Start(t *testing.T) {
	m.ConnectedSockets = make(chan *websocket.Conn, 1)

	upgrader := websocket.Upgrader{}

	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			conn *websocket.Conn
			err  error
		)

		if conn, err = upgrader.Upgrade(w, r, nil); err != nil {
			assert.FailNow(t, "failed to upgrade websocket connection")
		}

		// notify that a new websocket connection has been created
		m.ConnectedSockets <- conn
	}))

	url, _ := url.Parse(m.srv.URL)
	url.Scheme = "ws"
	m.WebsocketURL = url.String()
}

// Close will terminate the underlying server and release any resources.
func (m *MockServer) Close() {
	close(m.ConnectedSockets)
	m.srv.Close()
}

// TestClient is a WebSocket client that can attach itself to a server.
type TestClient struct {
	*websocket.Conn
	out chan WebSocketData
}

// Connect starts a new loop to process any messages being
// received by this client. Returns a channel which will contain
// any messages received.
func (c *TestClient) Connect(url string) <-chan WebSocketData {
	var (
		// out chan ChannelData
		err error
	)

	c.out = make(chan WebSocketData, 1)

	if c.Conn, _, err = websocket.DefaultDialer.Dial(url, nil); err != nil {
		panic(err)
	}

	c.SetPingHandler(func(appData string) error {
		c.out <- WebSocketData{
			messageType: websocket.PingMessage,
			data:        []byte(appData),
		}

		return nil
	})

	go func() {
		for {
			tt, mess, err := c.ReadMessage()
			if err != nil {
				close(c.out)
				c.out = nil
				return
			}
			c.out <- WebSocketData{tt, mess}
		}
	}()

	return c.out
}

// WaitReceive will wait the specified timeout period for a websocket message containing the
// given expectedType and expectedData.  If 'expectedData' is empty, it will be ignored.
func (c *TestClient) WaitReceive(expectedType int, dataRegex string) error {
	const timeout = time.Millisecond * 400

	if dataRegex == "" {
		dataRegex = ".*"
	}

	rxp := regexp.MustCompile(dataRegex)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout waiting for match for '%s'", dataRegex)
		case actual := <-c.out:
			if actual.messageType == expectedType && rxp.Match(actual.data) {
				return nil
			}
		}
	}
}
