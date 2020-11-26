package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ivan3bx/pickaxx"
)

func getServerManager(c *gin.Context) pickaxx.Manager {
	if m, ok := c.Get("manager"); ok {
		return m.(pickaxx.Manager)
	}
	c.AbortWithError(http.StatusInternalServerError, ErrSystem)
	return nil
}

func managerMiddleware(m pickaxx.Manager) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Set("manager", m)
	}
}
