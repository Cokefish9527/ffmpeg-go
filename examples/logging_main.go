package main

import (
	"fmt"
	"time"

	"github.com/u2takey/ffmpeg-go/utils"
)

func main() {
	fmt.Println("开始测试日志系统优化...")
	
	// 初始化日志系统
	utils.InitGlobalLogger()
	
	// 测试不同级别的日志
	testLogLevels()
	
	// 测试结构化日志
	testStructuredLogging()
	
	// 测试日志上下文
	testLogContext()
	
	fmt.Println("日志系统测试完成")
}

func testLogLevels() {
	fmt.Println("\n=== 测试不同级别的日志 ===")
	
	utils.Debug("这是一条DEBUG级别日志", nil)
	utils.Info("这是一条INFO级别日志", nil)
	utils.Warn("这是一条WARN级别日志", nil)
	utils.Error("这是一条ERROR级别日志", nil)
	
	// 测试动态调整日志级别
	fmt.Println("将日志级别调整为WARN...")
	utils.SetGlobalLoggerLevel(utils.WARN)
	utils.Debug("这条DEBUG日志不应该显示", nil)
	utils.Info("这条INFO日志不应该显示", nil)
	utils.Warn("这条WARN日志应该显示", nil)
	
	// 恢复日志级别
	utils.SetGlobalLoggerLevel(utils.INFO)
	utils.Info("日志级别已恢复", nil)
}

func testStructuredLogging() {
	fmt.Println("\n=== 测试结构化日志 ===")
	
	utils.Info("视频处理任务开始", map[string]string{
		"taskId":    "task-001",
		"inputFile": "input.mp4",
		"outputFile": "output.mp4",
		"width":     "1280",
		"height":    "720",
	})
	
	utils.Warn("视频处理进度较慢", map[string]string{
		"taskId":  "task-001",
		"progress": "35%",
		"elapsed":  "30s",
	})
	
	utils.Error("视频处理失败", map[string]string{
		"taskId": "task-001",
		"error":  "FFmpeg执行失败",
		"code":   "500",
	})
}

func testLogContext() {
	fmt.Println("\n=== 测试日志上下文 ===")
	
	// 模拟一个视频处理流程
	taskID := "task-002"
	
	utils.Info("开始处理视频任务", map[string]string{
		"taskId": taskID,
	})
	
	// 模拟解码阶段
	utils.Info("开始解码视频", map[string]string{
		"taskId": taskID,
		"stage":  "decoding",
	})
	time.Sleep(100 * time.Millisecond) // 模拟处理时间
	
	// 模拟编码阶段
	utils.Info("开始编码视频", map[string]string{
		"taskId": taskID,
		"stage":  "encoding",
	})
	time.Sleep(100 * time.Millisecond) // 模拟处理时间
	
	// 模拟完成
	utils.Info("视频处理完成", map[string]string{
		"taskId":  taskID,
		"stage":   "completed",
		"result":  "success",
	})
}