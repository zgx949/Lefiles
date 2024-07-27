package main

import (
	"Lefiles/config"
	"Lefiles/router"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB() // 初始化SQLite3

	r := gin.Default() // 初始化路由
	filesGroup := r.Group("/files")
	{
		router.FilesRouterInit(filesGroup)
	}

	r.Run(":8080")
}
