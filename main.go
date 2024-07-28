package main

import (
	"Lefiles/config"
	"Lefiles/router"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB() // 初始化SQLite3

	r := gin.Default()                         // 初始化路由
	r.LoadHTMLGlob("templates/*")              // 初始化html资源
	r.StaticFS("static", http.Dir("./static")) // 初始化静态资源
	filesGroup := r.Group("/files")
	{
		router.FilesRouterInit(filesGroup)
	}

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.Run(":8080")
}
