package cloud

import (
	"io/ioutil"
	"os"
	"witCon/log"
)

func ReadFile(path string) []byte {
	_, err := os.Stat(path)
	if err == nil {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			log.Error("read file err", "err", err)
			return nil
		} else {
			return b
		}
	} else {
		log.Error("nil file: %v", "err", err)
	}
	return nil
}

func WriteFile(path string, b []byte) {
	err := ioutil.WriteFile(path, b, 0777)
	if err != nil {
		log.Error("ioutils文件写入失败", "err", err)
		return
	}
	//fmt.Println("文件写入成功", err)
	return
}
