package minecraft

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsoleOutput(t *testing.T) {
	d := consoleEvent{"sample text"}

	bo, _ := json.Marshal(&d)

	assert.Equal(t, `{"output":"sample text"}`, string(bo))
	assert.Equal(t, "sample text", d.String())
}

func TestStateChangeEvent(t *testing.T) {
	d := stateChangeEvent{Running}

	bo, _ := json.Marshal(&d)

	assert.Equal(t, `{"status":"Running"}`, string(bo))
}

func TestPipeCommandOutput(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := exec.Command("cat")
		r, err := pipeCommandOutput(cmd)
		assert.NoError(t, err)
		assert.NotNil(t, r)
	})

	t.Run("fails when stdout set", func(t *testing.T) {
		cmd := exec.Command("cat")
		cmd.Stdout = &bytes.Buffer{}

		_, err := pipeCommandOutput(cmd)
		assert.Error(t, err)
	})

	t.Run("fails when stderr set", func(t *testing.T) {
		cmd := exec.Command("cat")
		cmd.Stderr = &bytes.Buffer{}

		_, err := pipeCommandOutput(cmd)
		assert.Error(t, err)
	})

}
