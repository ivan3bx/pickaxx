package minecraft

//go:generate stringer -type=ServerState -trimprefix=Server         -output=enums_string.go

//ServerState describes the current state of a Minecraft server.
type ServerState int

// Server Statuses
const (
	Unknown ServerState = iota
	Starting
	Running
	Stopping
	Stopped
)

// func (state ServerState) writeJSON(w io.Writer) error {
// 	jsonString := fmt.Sprintf(`{"status":"%s"}`, state.String())
// 	_, err := w.Write([]byte(jsonString))
// 	return err
// }
