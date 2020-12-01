package pickaxx

//go:generate stringer -type=ServerState -trimprefix=Server         -output=enums_string.go

//ServerState describes the current state of a Minecraft server.
type ServerState int

// Server Statuses
const (
	Unknown ServerState = iota
	Starting
	Started
	Stopping
	Stopped
)
