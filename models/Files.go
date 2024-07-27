package models

import (
	"gorm.io/gorm"
)

type FCB struct {
	gorm.Model
	Name     string `gorm:"not null"` // 文件或目录名称
	Size     uint   // 文件大小（对于目录可以为0）
	IsDir    bool   // 是否为目录
	ParentId uint   // 父目录ID
}

type Inode struct {
	gorm.Model
	FCBId     uint   `gorm:"not null"` // 关联的FCB ID
	FileIndex uint   // 文件索引号
	Url       string // 文件存储路径
	IsDeleted bool   // 是否被删除
}
