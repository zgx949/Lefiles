package local

import (
	"Lefiles/interfaces"
	"os"
)

type localBlockStorage struct{}

var LocalStorage interfaces.BlockStorage = localBlockStorage{}

// 读取文件块
func (localBlockStorage) ReadBlock(path string) ([]byte, error) {
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

// 写入文件块
func (localBlockStorage) WriteBlock(path string, block []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(block)
	if err != nil {
		return err
	}

	return nil
}
