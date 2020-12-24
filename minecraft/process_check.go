package minecraft

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/apex/log"
)

// ErrNoResponse is returned when a response is not provided within a certain period of time.
var ErrNoResponse = errors.New("no response or check failed")

// StatusNotifier handles a registery for observers of server state. This
// implementation can be accessed concurrently by multiple goroutines.
type StatusNotifier struct {
	sync.Mutex
	observers map[ServerState][]chan ServerState
}

// Register will register a new observer for the given states.
// Returns a new channel which will receive messages when the server changes
// to any of the state(s) provided.
func (n *StatusNotifier) Register(states ...ServerState) <-chan ServerState {
	n.Lock()
	defer n.Unlock()

	ch := make(chan ServerState, 10)

	if n.observers == nil {
		n.observers = make(map[ServerState][]chan ServerState)
	}

	for _, s := range states {
		if n.observers[s] == nil {
			n.observers[s] = []chan ServerState{}
		}
		n.observers[s] = append(n.observers[s], ch)
	}

	return ch
}

// Unregister will remove the given channel from the set of observers.
// The channel will be closed, and no further updates will be sent.
func (n *StatusNotifier) Unregister(ch <-chan ServerState) {
	n.Lock()
	defer n.Unlock()

	var target chan ServerState

	for key, v := range n.observers {
		copy := []chan ServerState{}

		for _, item := range v {
			if item != ch {
				copy = append(copy, item)
			} else {
				target = item
			}
		}

		n.observers[key] = copy
	}

	close(target)
}

// Notify will notify observers of the given parameters that
// the state has changed to the provided value.
func (n *StatusNotifier) Notify(st ServerState) {
	n.Lock()
	defer n.Unlock()

	for _, ch := range n.observers[st] {
		ch <- st
	}
}

// checkPort will continually check for 'liveness' on the given port on localhost.
// This loop will return in one of two cases:
//
// 1. If host does not respond (i.e. port is not open), returns an error.
// 2. If the provided channel receives a message, will quit (no error).
func checkPort(port int, cancel <-chan ServerState) error {

	time.Sleep(time.Second * 15) // initial delay

	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for {
		select {
		case <-cancel:
			return nil
		case <-ticker.C:
			if !portOpen("localhost", fmt.Sprintf("%d", port)) {
				log.WithField("action", "livenessProbe()").Warn("probe failed")
				return ErrNoResponse
			}
		}
	}
}

func portOpen(host string, port string) bool {
	hostname := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", hostname, time.Second)

	if err != nil || conn == nil {
		return false
	}

	conn.Close()
	return true
}
