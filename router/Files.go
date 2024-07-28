package router

import (
	"Lefiles/config"
	"Lefiles/models"
	"Lefiles/services"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

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

// 读取文件
func read(c *gin.Context) {
	id := c.Query("id")
	parseUint, err := strconv.ParseUint(id, 10, 32)
	if id == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	fcb, err := services.QueryFcbById(uint(parseUint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if fcb.IsDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read directory"})
		return
	}

	// 读取索引列表
	inodes, err := services.ReadInodes(fcb)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read inodes"})
		return
	}

	chunks := make([][]byte, len(inodes))
	var mutex sync.Mutex
	cond := sync.NewCond(&mutex)

	var wg sync.WaitGroup
	wg.Add(len(inodes))

	for index, inode := range inodes {
		go func(index int, inode models.Inode) {
			defer wg.Done()
			chunk, err := services.ReadChunkByUrl(inode.Url)
			if err != nil {
				chunk = nil
			}

			mutex.Lock()
			chunks[index] = chunk
			cond.Broadcast()
			mutex.Unlock()
		}(index, inode)
	}

	go func() {
		wg.Wait()
		mutex.Lock()
		cond.Broadcast()
		mutex.Unlock()
	}()

	filename := url.QueryEscape(fcb.Name)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+filename) // 用来指定下载下来的文件名
	c.Header("Content-Transfer-Encoding", "binary")

	for i := 0; i < len(chunks); i++ {
		mutex.Lock()
		for chunks[i] == nil {
			cond.Wait()
		}
		if chunks[i] != nil {
			_, err := c.Writer.Write(chunks[i])
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write chunk"})
				mutex.Unlock()
				return
			}
			chunks[i] = nil // 尽快释放内存
		}
		mutex.Unlock()
	}
}

// 删除文件/文件夹
func del(c *gin.Context) {
	id := c.Query("id")
	parseUint, err := strconv.ParseUint(id, 10, 32)
	if id == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	rootId := uint(parseUint)
	_, err = services.QueryFcbById(rootId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File/Directory not found"})
		return
	}

	// 广度优先搜索删除队列
	queue := []uint{rootId}

	for len(queue) > 0 {
		currentId := queue[0]
		queue = queue[1:]

		currentFcb, err := services.QueryFcbById(currentId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !currentFcb.IsDir {
			// 删除文件FCB
			if err := config.DB.Delete(&currentFcb).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 将对应的索引节点标记为已删除
			inodes, err := services.ReadInodes(currentFcb)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot read inodes"})
				return
			}

			for _, inode := range inodes {
				// 软删除
				if err := config.DB.Delete(&inode).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		} else {
			// 获取子文件夹，将其 ID 加入队列
			subFCBs, err := services.QueryFcbByParentId(currentId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			for _, subFcb := range subFCBs {
				queue = append(queue, subFcb.ID)
			}

			// 删除文件夹
			if err := config.DB.Delete(&currentFcb).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

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

const chunkSize = 1024 * 1024 // 1MB

// 上传文件
func upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	// 获取对应路径的FCB
	paths := strings.Split(path, "/")
	dirPath := paths[:len(paths)-1]
	fileName := paths[len(paths)-1]

	pathFcb, err := services.FindPathFCB(strings.Join(dirPath, "/"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 新建FCB
	var newFCB models.FCB
	newFCB.Name = fileName
	newFCB.ParentId = pathFcb.ID
	newFCB.IsDir = false
	newFCB.Size = uint(header.Size)

	// 检查FCB是否存在
	var existingFCB models.FCB
	if err = config.DB.Where("name = ? AND parent_id = ?", fileName, pathFcb.ID).First(&existingFCB).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "File already exists"})
		return
	}

	if err = config.DB.Create(&newFCB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取文件大小
	totalSize := header.Size
	totalChunks := (totalSize + int64(chunkSize) - 1) / int64(chunkSize)

	// 分配inode索引节点
	inodes, err := services.GetInodes(uint(totalChunks), "baidu", newFCB.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var currentChunk int64 = 0
	buf := make([]byte, chunkSize)

	for {
		n, err := file.Read(buf)
		if err != nil {
			break
		}
		err = services.WriteBlockByUrl(inodes[currentChunk].Url, buf[:n])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		currentChunk++
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
}

// 下载文件
func download(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	parseUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	fcb, err := services.QueryFcbById(uint(parseUint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if fcb.IsDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot download directory"})
		return
	}

	inodes, err := services.ReadInodes(fcb)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read inodes"})
		return
	}

	for i := range inodes {
		chunk, err := services.ReadChunkByUrl(inodes[i].Url)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = c.Writer.Write(chunk)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write chunk"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "File downloaded successfully"})
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
	rg.POST("/upload", upload)
	rg.GET("/download", download)
}
