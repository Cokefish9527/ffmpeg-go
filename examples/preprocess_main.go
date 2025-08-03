package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/u2takey/ffmpeg-go/service"
)

func main() {
	// 创建视频信息缓存
	cache := service.NewVideoInfoCache()
	
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
	
	fmt.Println("开始测试输入文件预处理...")
	
	// 第一次分析（无缓存）
	startTime := time.Now()
	for i, file := range inputFiles {
		fullPath := filepath.Join(wd, "video", file)
		
		fmt.Printf("分析文件 %d/%d: %s\n", i+1, len(inputFiles), file)
		
		info, err := cache.AnalyzeVideo(fullPath)
		if err != nil {
			fmt.Printf("分析文件 %s 失败: %v\n", file, err)
			continue
		}
		
		fmt.Printf("  文件名: %s\n", info.FileName)
		fmt.Printf("  文件大小: %.2f MB\n", float64(info.FileSize)/(1024*1024))
		fmt.Printf("  时长: %.2f 秒\n", info.Duration)
		fmt.Printf("  编码: %s\n", info.Codec)
		fmt.Printf("  分辨率: %dx%d\n", info.Width, info.Height)
		fmt.Printf("  FPS: %.2f\n", info.FPS)
		fmt.Printf("  比特率: %d kbps\n", info.Bitrate/1000)
		fmt.Println()
	}
	
	firstAnalysisTime := time.Since(startTime)
	fmt.Printf("第一次分析耗时: %v\n", firstAnalysisTime)
	
	// 第二次分析（使用缓存）
	startTime = time.Now()
	for i, file := range inputFiles {
		fullPath := filepath.Join(wd, "video", file)
		
		fmt.Printf("缓存分析文件 %d/%d: %s\n", i+1, len(inputFiles), file)
		
		info, err := cache.AnalyzeVideo(fullPath)
		if err != nil {
			fmt.Printf("分析文件 %s 失败: %v\n", file, err)
			continue
		}
		
		fmt.Printf("  文件名: %s\n", info.FileName)
		fmt.Printf("  文件大小: %.2f MB\n", float64(info.FileSize)/(1024*1024))
		fmt.Printf("  时长: %.2f 秒\n", info.Duration)
		fmt.Printf("  编码: %s\n", info.Codec)
		fmt.Printf("  分辨率: %dx%d\n", info.Width, info.Height)
		fmt.Printf("  FPS: %.2f\n", info.FPS)
		fmt.Printf("  比特率: %d kbps\n", info.Bitrate/1000)
		fmt.Println()
	}
	
	secondAnalysisTime := time.Since(startTime)
	fmt.Printf("缓存分析耗时: %v\n", secondAnalysisTime)
	
	// 计算性能提升
	improvement := float64(firstAnalysisTime.Nanoseconds()-secondAnalysisTime.Nanoseconds()) / float64(firstAnalysisTime.Nanoseconds()) * 100
	fmt.Printf("性能提升: %.2f%%\n", improvement)
	
	// 测试预处理功能
	fmt.Println("\n测试预处理功能...")
	preprocessStart := time.Now()
	processedFiles, err := cache.PreprocessInputFiles(inputFiles, wd)
	if err != nil {
		fmt.Printf("预处理失败: %v\n", err)
		return
	}
	preprocessTime := time.Since(preprocessStart)
	
	fmt.Printf("预处理完成，耗时: %v\n", preprocessTime)
	fmt.Printf("处理后的文件列表:\n")
	for i, file := range processedFiles {
		fmt.Printf("  %d. %s\n", i+1, file)
	}
}