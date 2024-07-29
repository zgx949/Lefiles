package storages

import (
	"Lefiles/interfaces"
	"Lefiles/services/storages/baidu"
	"Lefiles/services/storages/local"
)

// 存储协议映射
var PROTMAP = map[string]interfaces.BlockStorage{
	"local": local.LocalStorage,
	"baidu": baidu.BaiduStorage,
}
