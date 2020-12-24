package minecraft

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsoleOutput(t *testing.T) {
	d := consoleOutput{"sample text"}

	bo, _ := json.Marshal(&d)

	assert.Equal(t, `{"output":"sample text"}`, string(bo))
	assert.Equal(t, "sample text", d.String())
}

func TestStateChangeEvent(t *testing.T) {
	d := stateChangeEvent{Running}

	bo, _ := json.Marshal(&d)

	assert.Equal(t, `{"status":"Running"}`, string(bo))
}
