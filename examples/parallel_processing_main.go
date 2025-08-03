package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/u2takey/ffmpeg-go/service"
)

func main() {
	// 创建测试用的Worker
	taskQueue := service.NewInMemoryTaskQueue()
	worker := service.NewWorker(taskQueue)
	
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("无法获取当前工作目录: %v\n", err)
		return
	}
	
	// 定义输入文件
	inputFiles := []string{
		"1.ts",
		"2.ts",
		"3.ts",
		"4.ts",
		"5.ts",
	}
	
	fmt.Println("开始测试并行解码功能...")
	
	// 测试并行解码
	startTime := time.Now()
	decodedFiles, err := worker.ParallelDecodeForTest(inputFiles, wd)
	if err != nil {
		fmt.Printf("并行解码失败: %v\n", err)
		return
	}
	
	decodeTime := time.Since(startTime)
	fmt.Printf("并行解码完成，耗时: %v\n", decodeTime)
	
	fmt.Printf("解码后的文件列表:\n")
	for i, file := range decodedFiles {
		// 检查文件是否存在
		if _, err := os.Stat(file); err == nil {
			// 获取文件信息
			fileInfo, _ := os.Stat(file)
			fmt.Printf("  %d. %s (大小: %.2f MB)\n", i+1, filepath.Base(file), float64(fileInfo.Size())/(1024*1024))
		} else {
			fmt.Printf("  %d. %s (文件不存在)\n", i+1, filepath.Base(file))
		}
	}
	
	// 清理临时文件
	if len(decodedFiles) > 0 {
		tempDir := filepath.Dir(decodedFiles[0])
		os.RemoveAll(tempDir)
		fmt.Printf("\n已清理临时目录: %s\n", tempDir)
	}
}