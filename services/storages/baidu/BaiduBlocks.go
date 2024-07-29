package baidu

import (
	"Lefiles/interfaces"
	"Lefiles/services/storages/baidu/pcscommand"
	pcsconfig2 "Lefiles/services/storages/baidu/pcsconfig"
	"Lefiles/services/storages/baidu/pcsfunctions/pcsdownload"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type baiduBlockStorage struct{}

var (
	downloadMode pcsdownload.DownloadMode
	BaiduStorage interfaces.BlockStorage = baiduBlockStorage{}
	baidu        *pcsconfig2.Baidu
	do           = &pcscommand.DownloadOptions{}
)

func init() {
	//budss := os.Getenv("BDUSS")
	//var err error
	//baidu, err = pcsconfig2.Config.SetupUserByBDUSS(budss, "", "", "")
	//if err != nil {
	//	log.Fatal("百度网盘登陆错误：", err.Error())
	//	return
	//}
	//log.Default().Println("百度帐号登录成功:", baidu.Name)
	err := pcsconfig2.Config.Init()
	switch err {
	case nil:
	case pcsconfig2.ErrConfigFileNoPermission, pcsconfig2.ErrConfigContentsParseError:
		fmt.Fprintf(os.Stderr, "FATAL ERROR: config file error: %s\n", err)
		os.Exit(1)
	default:
		fmt.Printf("WARNING: config init error: %s\n", err)
	}
}

// 读取文件块
func (baiduBlockStorage) ReadBlock(path string) ([]byte, error) {
	// 建立一个io.Reader来存储block
	opt := &pcscommand.LocateDownloadOption{
		FromPan: false,
	}
	url, err := pcscommand.RunLocateDownload(path, opt)
	if err != nil {
		log.Fatal("读取百度网盘块错误：", err.Error())
		return nil, err
	}

	// 执行下载文件块
	resp, err := http.Get(url.String())
	if err != nil {
		log.Fatal("下载文件块错误：", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("下载文件块失败，状态码: %d", resp.StatusCode)
		return nil, fmt.Errorf("下载文件块失败，状态码: %d", resp.StatusCode)
	}

	block, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("读取文件块错误：", err.Error())
		return nil, err
	}

	return block, nil
}

// WriteBlock 写入文件块
func (baiduBlockStorage) WriteBlock(path string, block []byte) (err error) {
	opt := &pcscommand.UploadOptions{
		Parallel:        4,
		MaxRetry:        3,
		Load:            1,
		NoRapidUpload:   false,
		NoSplitFile:     false,
		Policy:          "fail",
		NoFilenameCheck: false,
	}

	// 生成临时文件
	tempFilePath := "./temp" + strings.Split(path, ".")[1]
	err = ioutil.WriteFile(tempFilePath, block, os.ModePerm)
	if err != nil {
		fmt.Printf("写入临时文件错误: %s\n", err)
		return err
	}
	defer os.Remove(tempFilePath)

	pcscommand.RunUpload([]string{tempFilePath}, path, opt)
	return nil
}
