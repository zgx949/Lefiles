package services

import (
	"Lefiles/config"
	"Lefiles/models"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// QueryFcb 通过名称和上级id搜索FCB
func QueryFcb(name string, parentId uint) (fcb models.FCB, err error) {
	err = config.
		DB.Where("name = ? AND parent_id = ?", name, parentId).
		First(&fcb).Error
	return
}

// QueryFcbByParentId 通过pid找FCBs
func QueryFcbByParentId(parentId uint) (fcbs []models.FCB, err error) {
	err = config.
		DB.Where("parent_id = ?", parentId).Order("is_dir desc, name").
		Find(&fcbs).Error
	return
}

// QueryFcbById 通过名称和上级id搜索FCB
func QueryFcbById(id uint) (fcb models.FCB, err error) {
	err = config.
		DB.Where("id = ?", id).
		First(&fcb).Error
	return
}

func FindPathFCB(path string) (fcb models.FCB, err error) {
	if path == "" {
		err = filepath.ErrBadPattern
		return
	}
	// TODO: 读取PATH的缓存信息

	paths := strings.Split(path, "/")
	currentPath := ""
	// 从根目录开始逐层查询
	var parentId uint

	for _, p := range paths {
		if p == "" {
			continue
		}
		if fcb, err = QueryFcb(p, parentId); err != nil {
			err = filepath.ErrBadPattern
			return
		} else {
			parentId = fcb.ID
			currentPath += "/" + p
			// TODO: 缓存目录FCB信息
		}
	}
	return
}

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
