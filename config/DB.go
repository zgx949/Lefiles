package config

import (
	"Lefiles/models"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("Lefiles.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	// 自动迁移模型
	err = DB.AutoMigrate(&models.FCB{}, &models.Inode{})
	if err != nil {
		log.Fatal("failed to migrate database", err)
	}
}
