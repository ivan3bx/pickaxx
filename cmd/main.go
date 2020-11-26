package main

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/gin-gonic/gin"
	"github.com/ivan3bx/pickaxx"
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetHandler(cli.Default)

	// start the server manager
	manager := pickaxx.NewServerManager()
	go manager.Run()

	// start web server
	e := gin.Default()
	e.Static("/assets", "public")
	e.LoadHTMLFiles("templates/index.html")

	routes := e.Group("/", managerMiddleware(manager))
	{
		routes.GET("/", rootHandler)
		routes.GET("/ws", webSocketHandler)
		routes.POST("/start", startServerHandler)
		routes.POST("/stop", stopServerHandler)
	}

	e.Run("127.0.0.1:8080")
}
