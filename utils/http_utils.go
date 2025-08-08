package utils

import (
	"net/http"
	"os"
)

// DownloadFile 下载文件到指定路径
func DownloadFile(url, filepath string) error {
	// 发起HTTP GET请求
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 创建目标文件
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 将HTTP响应内容写入文件
	_, err = out.ReadFrom(resp.Body)
	return err
}