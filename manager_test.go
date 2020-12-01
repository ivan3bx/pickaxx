package pickaxx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerManager(t *testing.T) {
	w := strings.Builder{}
	m := NewServerManager(&w).(*serverManager)

	t.Run("state initialized", func(t *testing.T) {
		assert.Equal(t, Unknown, m.state)
	})

}
