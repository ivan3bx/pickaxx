package pickaxx

//go:generate stringer -type=ServerState -trimprefix=Server         -output=enums_string.go

import "sync"

//ServerState describes the current state of a Minecraft server.
type ServerState int

// Server Statuses
const (
	Starting ServerState = iota
	Started
	Stopping
	Stopped
	Unknown
)

// ManagedState is a placeholder..
type ManagedState struct {
	sync.Mutex
	current ServerState
	update  chan ServerState
}
