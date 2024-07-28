package services

import (
	"Lefiles/config"
	"Lefiles/interfaces"
	"Lefiles/models"
	"Lefiles/services/storages"
	"Lefiles/services/storages/baidu"
	"Lefiles/utils"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func ReadInodes(fcb models.FCB) (inodes []models.Inode, err error) {
	// 检查 FCB 是否存在
	if fcb.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// 查询与 FCB 相关的 Inodes
	if err = config.DB.Where("fcb_id = ?", fcb.ID).Find(&inodes).Error; err != nil {
		return nil, err
	}

	return inodes, nil
}

// 存储协议映射
var protMap = map[string]interfaces.BlockStorage{
	"local": storages.LocalStorage,
	"baidu": baidu.BaiduStorage,
}

func ReadChunkByUrl(url string) ([]byte, error) {
	// 解析不同url协议，然后从本地或者远程读取文件块并返回
	items := strings.Split(url, "://")
	if len(items) != 2 {
		return nil, errors.New("invalid URL format")
	}

	prot, path := items[0], items[1]
	if storage, ok := protMap[prot]; ok {
		path = "./blocks/" + path
		block, err := storage.ReadBlock(path)
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	log.Println("Unsupported protocol:", prot, "Path:", path)
	return nil, errors.New("unsupported protocol")
}

func WriteBlockByUrl(url string, buf []byte) error {
	// 解析不同url协议，然后存储文件块
	items := strings.Split(url, "://")
	if len(items) != 2 {
		return errors.New("invalid URL format")
	}

	prot, path := items[0], items[1]
	if storage, ok := protMap[prot]; ok {
		path = "./blocks/" + path
		if err := storage.WriteBlock(path, buf); err != nil {
			return err
		}
		return nil
	}

	log.Println("Unsupported protocol:", prot, "Path:", path)
	return errors.New("unsupported protocol")
}

// GetInodes 新建索引节点
func GetInodes(amount uint, prot string, fcbId uint) ([]models.Inode, error) {
	var (
		inodes        []models.Inode
		deletedInodes []models.Inode
		mutex         sync.Mutex
	)

	// 加锁
	mutex.Lock()
	defer mutex.Unlock()

	// 查询已删除的节点
	if err := config.DB.Unscoped().
		Where("deleted_at is not null AND url like ?", prot).
		Limit(int(amount)).
		Find(&deletedInodes).Error; err != nil {
		return nil, err
	}

	// 使用已删除的节点
	for i := 0; i < len(deletedInodes) && uint(len(inodes)) < amount; i++ {
		inode := &deletedInodes[i] // 获取指针
		inode.FCBId = fcbId
		inode.FileIndex = uint(i)
		inode.DeletedAt = gorm.DeletedAt{} // 重置删除时间，复用节点
		if err := config.DB.Save(inode).Error; err != nil {
			return nil, err
		}
		inodes = append(inodes, *inode)
	}

	// 如果已删除节点不足，则新建剩余的节点
	for i := uint(len(inodes)); i < amount; i++ {
		inode := models.Inode{
			Url:       prot + "://" + utils.GenerateUUID(),
			FCBId:     fcbId,
			FileIndex: i,
		}
		if err := config.DB.Create(&inode).Error; err != nil {
			return nil, err
		}
		inodes = append(inodes, inode)
	}

	return inodes, nil
}
