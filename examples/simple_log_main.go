package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/u2takey/ffmpeg-go/utils"
)

func main() {
	fmt.Println("测试日志系统...")
	
	// 获取当前目录
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取当前目录失败: %v\n", err)
		return
	}
	
	// 设置日志目录
	logDir := filepath.Join(currentDir, "log")
	fmt.Printf("日志目录: %s\n", logDir)
	
	// 创建日志记录器
	logger, err := utils.NewLogger(logDir, "test_log", utils.DEBUG, 1024*1024, 5)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()
	
	// 测试日志记录
	logger.Debug("这是一条DEBUG日志", map[string]string{"key": "value"})
	logger.Info("这是一条INFO日志", map[string]string{"module": "test"})
	logger.Warn("这是一条WARN日志", map[string]string{"warning": "test warning"})
	logger.Error("这是一条ERROR日志", map[string]string{"error": "test error"})
	
	fmt.Println("日志测试完成")
	
	// 检查日志文件是否存在
	logFile := filepath.Join(logDir, "test_log.log")
	if _, err := os.Stat(logFile); err == nil {
		fmt.Printf("日志文件已创建: %s\n", logFile)
		
		// 读取并显示日志内容
		content, err := os.ReadFile(logFile)
		if err != nil {
			fmt.Printf("读取日志文件失败: %v\n", err)
		} else {
			fmt.Printf("日志内容:\n%s\n", string(content))
		}
	} else {
		fmt.Printf("日志文件不存在: %v\n", err)
	}
}