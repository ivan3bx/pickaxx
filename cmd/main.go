package main

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ivan3bx/pickaxx"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetHandler(cli.Default)

	serverManager := pickaxx.NewServerManager()
	go serverManager.Run()

	r := gin.Default()
	r.Static("/assets", "public")
	r.LoadHTMLFiles("templates/index.html")

	r.GET("/", func(c *gin.Context) {
		var lines []string

		if serverManager.Active() {
			content, _ := ioutil.ReadFile("testserver/logs/latest.log")
			lines = strings.Split(string(content), "\n")
		}
		c.HTML(http.StatusOK, "index.html", gin.H{"logLines": lines})
	})

	r.GET("/ws", func(c *gin.Context) {
		serveWs(c, serverManager)
	})

	r.POST("/start", func(c *gin.Context) {
		err := serverManager.StartServer()

		if err == pickaxx.ErrProcessStart {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"err": "server already running",
			})
			return
		}

		if err != nil {
			log.WithError(err).Error("server failed to start")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"err": "server error"})
		}
	})

	r.POST("/stop", func(c *gin.Context) {
		serverManager.StopServer()
	})

	r.Run("127.0.0.1:8080")
}

func serveWs(c *gin.Context, manager pickaxx.Manager) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	manager.AddClient(conn)
}
