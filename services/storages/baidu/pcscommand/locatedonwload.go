package pcscommand

import (
	"log"
	"net/url"
)

// RunLocateDownload 执行获取直链
func RunLocateDownload(pcspaths string, opt *LocateDownloadOption) (url *url.URL, err error) {
	if opt == nil {
		opt = &LocateDownloadOption{}
	}

	absPaths, err := matchPathByShellPattern(pcspaths[1:])
	if err != nil {
		log.Default().Fatal(err.Error())
		return nil, err
	}

	pcs := GetBaiduPCS()

	pcspath := absPaths[0]
	info, err := pcs.LocateDownload(pcspath)
	if err != nil {
		log.Default().Printf("[%d] %s, 路径: %s\n", 0, err, pcspath)
		return nil, err
	}

	// 返回下载地址
	return info.SingleURL(true), nil

}
