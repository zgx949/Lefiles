package utils

import "github.com/google/uuid"

// GenerateUUID 生成一个不重复的UUID
func GenerateUUID() string {
	return uuid.New().String()
}
