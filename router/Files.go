package router

import (
	"Lefiles/config"
	"Lefiles/models"
	"Lefiles/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme/autocert"
)

var (
	dirCache autocert.Cache
)

// 打开文件
func open(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	paths := strings.Split(path, "/")

	dirPath := paths[:len(paths)-1]

	fcb, err := services.FindPathFCB(strings.Join(dirPath, "/"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fcb)
}

// 打开文件夹
func ls(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}
	// TODO: 读取PATH的缓存信息
	dirFcb, err := services.FindPathFCB(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fcbs, err := services.QueryFcbByParentId(dirFcb.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fcbs)
}

// TODO:读取文件
func read(c *gin.Context) {
	id := c.Query("id")
	parseUint, err := strconv.ParseUint(id, 10, 32)
	if id == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	if fcb, err := services.QueryFcbById(uint(parseUint)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	} else {
		if fcb.IsDir {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read directory"})
			return
		}
		// TODO: 读取索引列表，并返回文件流

	}

}

// 删除文件/文件夹
func del(c *gin.Context) {
	var err error
	id := c.Query("id")
	parseUint, err := strconv.ParseUint(id, 10, 32)
	if id == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	var fcb models.FCB

	if fcb, err = services.QueryFcbById(uint(parseUint)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File/Directory not found"})
		return
	} else {
		if err := config.DB.Delete(&fcb).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// TODO: 需要迭代删除, 队列 广度优先删除
	//fcbs, err := services.QueryFcbByParentId(fcb.ID)
	//for _, fcb := range fcbs {
	//
	//}
	c.JSON(http.StatusOK, gin.H{"message": "File/Directory deleted successfully"})
}

// 更新文件/文件夹FCB
func update(c *gin.Context) {
	var updatedFCB models.FCB
	if err := c.BindJSON(&updatedFCB); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	var fcb models.FCB
	if err := config.DB.First(&fcb, updatedFCB.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File/Directory not found"})
		return
	}

	fcb.Name = updatedFCB.Name
	//fcb.Size = updatedFCB.Size
	//fcb.IsDir = updatedFCB.IsDir
	fcb.ParentId = updatedFCB.ParentId

	if err := config.DB.Save(&fcb).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "FCB updated successfully", "fcb": fcb})
}

// 创建文件FCB
func create(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	paths := strings.Split(path, "/")
	dirPath := paths[:len(paths)-1]
	fileName := paths[len(paths)-1]

	pathFcb, err := services.FindPathFCB(strings.Join(dirPath, "/"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var newFCB models.FCB
	newFCB.Name = fileName
	newFCB.ParentId = pathFcb.ID
	newFCB.IsDir = false

	var existingFCB models.FCB
	if err = config.DB.Where("name = ? AND parent_id = ?", fileName, pathFcb.ID).First(&existingFCB).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "File already exists"})
		return
	}

	if err = config.DB.Create(&newFCB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File created successfully", "fcb": newFCB})
}

// 创建文件夹FCB
func mkdir(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	paths := strings.Split(path, "/")

	// 从根目录开始逐层创建
	var parentId uint
	for i, p := range paths {
		if p == "" {
			continue
		}
		var fcb models.FCB
		if err := config.DB.Where("name = ? AND parent_id = ?", p, parentId).First(&fcb).Error; err != nil {
			// 如果没有找到，则创建新的目录
			newFCB := models.FCB{
				Name:     p,
				ParentId: parentId,
				IsDir:    true,
			}
			if err := config.DB.Create(&newFCB).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			fcb = newFCB
		}
		parentId = fcb.ID
		if i == len(paths)-1 && !fcb.IsDir {
			c.JSON(http.StatusConflict, gin.H{"error": "File with the same name already exists"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Directory created successfully"})
}

// FilesRouterInit 初始化文件路由
func FilesRouterInit(rg *gin.RouterGroup) {
	rg.GET("/open", open)
	rg.GET("/ls", ls)
	rg.GET("/read", read)
	rg.DELETE("/del", del)
	rg.PUT("/update", update)
	rg.POST("/create", create)
	rg.POST("/mkdir", mkdir)
}
