package pickaxx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerManager(t *testing.T) {
	m := &ProcessManager{}

	t.Run("state initialized", func(t *testing.T) {
		assert.Equal(t, Unknown.String(), m.state.String())
	})

}
