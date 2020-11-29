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

	// start a new manager
	clientManager := pickaxx.ClientManager{}
	manager := pickaxx.NewServerManager(&clientManager)

	// start web server
	e := gin.New()
	e.Use(gin.Logger(), gin.Recovery())

	e.Static("/assets", "public")
	e.LoadHTMLFiles("templates/index.html")

	e.GET("/ws", webSocketHandler(&clientManager))

	routes := e.Group("/", managerMiddleware(manager))
	{
		routes.GET("/", rootHandler)
		routes.POST("/start", startServerHandler)
		routes.POST("/stop", stopServerHandler)
	}

	e.Run("127.0.0.1:8080")
}
