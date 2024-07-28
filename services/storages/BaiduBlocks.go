package storages

import (
	"Lefiles/interfaces"
	"bytes"
	"log"
	"os"

	bdyp "github.com/zcxey2911/bdyp_upload_golang"
)

type baiduBlockStorage struct{}

var BaiduStorage interfaces.BlockStorage = baiduBlockStorage{}
var bcloud = bdyp.Bcloud{}

func init() {
	res, err := bcloud.GetToken("obb获取的code", "oob", "应用appkey", "应用appsecret")
	if err != nil {
		log.Default().Fatal("err", err)
	} else {
		log.Default().Println("接口的token是: %#v\n", res.AccessToken)
	}
}

// 读取文件块
func (baiduBlockStorage) ReadBlock(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	chunk := make([]byte, stat.Size())
	_, err = file.Read(chunk)
	if err != nil {
		return nil, err
	}

	return chunk, nil
}

// WriteBlock 写入文件块
func (baiduBlockStorage) WriteBlock(path string, block []byte) error {
	// 建立一个io.Reader来存储block
	file := bytes.NewReader(block)

	// 上传文件
	err := bcloud.Upload(&bdyp.FileUploadReq{
		Name:  path,
		File:  file,
		RType: nil,
	})
	return err
}
