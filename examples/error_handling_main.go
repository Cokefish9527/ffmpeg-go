package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	fmt.Println("开始测试错误处理和恢复机制...")
	
	// 测试FFmpeg执行错误处理
	testFFmpegErrorHandling()
	
	// 测试文件不存在错误处理
	testFileNotFoundErrorHandling()
	
	// 测试重试机制
	testRetryMechanism()
}

func testFFmpegErrorHandling() {
	fmt.Println("\n=== 测试FFmpeg执行错误处理 ===")
	
	// 创建一个会导致错误的FFmpeg命令（无效参数）
	cmd := exec.Command("ffmpeg", "-invalid", "parameter")
	
	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("FFmpeg错误处理测试成功\n")
		fmt.Printf("错误信息: %v\n", err)
		fmt.Printf("执行时间: %v\n", duration)
		if len(output) > 0 {
			fmt.Printf("输出内容: %s\n", string(output))
		}
	} else {
		fmt.Printf("FFmpeg错误处理测试失败：命令意外成功执行\n")
	}
}

func testFileNotFoundErrorHandling() {
	fmt.Println("\n=== 测试文件不存在错误处理 ===")
	
	// 尝试处理一个不存在的文件
	cmd := exec.Command("ffmpeg", "-i", "nonexistent_file.mp4", "output.mp4")
	
	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("文件不存在错误处理测试成功\n")
		fmt.Printf("错误信息: %v\n", err)
		fmt.Printf("执行时间: %v\n", duration)
		if len(output) > 0 {
			fmt.Printf("输出内容: %s\n", string(output))
		}
	} else {
		fmt.Printf("文件不存在错误处理测试失败：命令意外成功执行\n")
	}
	
	// 清理可能创建的文件
	os.Remove("output.mp4")
}

func testRetryMechanism() {
	fmt.Println("\n=== 测试重试机制 ===")
	
	// 模拟一个可能失败的任务，测试重试机制
	failures := 0
	maxRetries := 3
	
	fmt.Printf("模拟任务执行，最多重试%d次\n", maxRetries)
	
	for i := 0; i <= maxRetries; i++ {
		// 模拟一个有时会失败的任务
		if simulateTask(&failures) {
			fmt.Printf("第%d次尝试成功\n", i+1)
			return
		} else {
			fmt.Printf("第%d次尝试失败\n", i+1)
			if i < maxRetries {
				fmt.Printf("等待%d秒后重试...\n", (i+1)*2)
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
			}
		}
	}
	
	fmt.Printf("任务最终失败，已重试%d次\n", maxRetries)
}

// simulateTask 模拟一个可能失败的任务
func simulateTask(failures *int) bool {
	*failures++
	
	// 模拟前两次失败，第三次成功
	if *failures <= 2 {
		return false
	}
	
	return true
}