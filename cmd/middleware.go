package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ivan3bx/pickaxx"
)

const contextKey = "processManager"

func getServerManager(c *gin.Context) pickaxx.ServerManager {
	if m, ok := c.Get(contextKey); ok {
		return m.(pickaxx.ServerManager)
	}
	c.AbortWithError(http.StatusInternalServerError, ErrSystem)
	return nil
}

func managerMiddleware(m pickaxx.ServerManager) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Set(contextKey, m)
	}
}
