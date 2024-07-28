package main

import (
	"Lefiles/config"
	"Lefiles/router"
	"Lefiles/services/storages/baidu/pcsconfig"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func init() {

	err := pcsconfig.Config.Init()
	switch err {
	case nil:
	case pcsconfig.ErrConfigFileNoPermission, pcsconfig.ErrConfigContentsParseError:
		fmt.Fprintf(os.Stderr, "FATAL ERROR: config file error: %s\n", err)
		os.Exit(1)
	default:
		fmt.Printf("WARNING: config init error: %s\n", err)
	}
}
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
