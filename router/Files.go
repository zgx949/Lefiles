package router

import (
	"Lefiles/config"
	"Lefiles/models"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// 创建文件夹或文件
func createFCB(c *gin.Context, isDir bool) {
	var newFCB models.FCB

	// 绑定 JSON 数据到 FCB 结构体
	if err := c.BindJSON(&newFCB); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	newFCB.IsDir = isDir
	// 检查对应当前目录下是否有重复文件名
	var existingFCB models.FCB
	if err := config.DB.Where("name = ? AND parent_id = ?", newFCB.Name, newFCB.ParentId).First(&existingFCB).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "File/Directory already exists"})
		return
	}

	// 创建新的 FCB 实体并保存到数据库
	if err := config.DB.Create(&newFCB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回成功消息
	c.JSON(http.StatusOK, gin.H{"message": "FCB created successfully", "fcb": newFCB})
}

// 打开目录
func openDir(c *gin.Context) {
	parentId := c.Query("parent_id")
	query := config.DB

	if parentId == "" {
		query = query.Where("parent_id = ''").Or("parent_id is null")
	} else {
		query = query.Where("parent_id = ?", parentId)
	}

	var fcbs []models.FCB
	query.Find(&fcbs)

	c.JSON(http.StatusOK, fcbs)
}

// TODO: 下载文件
func readFile(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	var inode models.Inode
	if err := config.DB.First(&inode, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.File(inode.Url)
}

// 创建文件夹
func createDir(c *gin.Context) {
	createFCB(c, true)
}

// TODO: 上传文件
func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	destPath := filepath.Join("./uploads", filepath.Base(file.Filename))
	if err := c.SaveUploadedFile(file, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newInode := models.Inode{
		Url: destPath,
		// 其他字段根据需求设置
	}

	if err := config.DB.Create(&newInode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
}

// TODO: 删除文件/文件夹
func deleteFCB(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	// 查找指定 ID 的记录
	var fcb models.FCB
	if err := config.DB.First(&fcb, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 从数据库中删除记录 TODO：同时标记索引节点为删除状态，以及删除子目录（广度有限搜索）
	if err := config.DB.Delete(&fcb).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File/Directory deleted successfully"})
}

// FilesRouterInit 初始化文件路由
func FilesRouterInit(rg *gin.RouterGroup) {
	rg.GET("/openDir", openDir)
	rg.GET("/readFile", readFile)
	rg.POST("/createDir", createDir)
	rg.POST("/uploadFile", uploadFile)
	rg.DELETE("/deleteFCB", deleteFCB)
}
