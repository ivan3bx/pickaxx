package pickaxx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerManager(t *testing.T) {
	w := strings.Builder{}
	m := NewServerManager(&w).(*serverManager)

	t.Run("channels initialized", func(t *testing.T) {
		assert.NotNil(t, m.stopRunLoop)
	})

	t.Run("state initialized", func(t *testing.T) {
		assert.Equal(t, Unknown, m.state.current)
		assert.NotNil(t, m.state.update)
	})

}
