package main

import (
    "os"
    "net/http"


	"github.com/gin-gonic/gin"
)

var (
	listenOn   = os.Getenv("LISTEN_ON")
)

func main() {
    r := gin.Default()
    r.GET("/ping", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "pong",
        })
    })
    r.Run(listenOn) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
