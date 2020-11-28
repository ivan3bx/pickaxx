package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ivan3bx/pickaxx"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestManagerMiddleware(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	t.Run("middleware is set", func(t *testing.T) {
		expected := pickaxx.NewServerManager()
		mwFunc := managerMiddleware(expected)

		mwFunc(c)

		actual := getServerManager(c)
		assert.Same(t, expected, actual)
	})
}
