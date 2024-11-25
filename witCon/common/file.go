package common

import (
	"io/ioutil"
	"os"
)

// 将文件转换为字节切片
func FileToBytes(filename string) ([]byte, error) {
	// 打开文件
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取文件内容
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// 将字节切片转换为文件
func BytesToFile(bytes []byte, filename string) error {
	// 创建文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入字节切片到文件
	_, err = file.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}
