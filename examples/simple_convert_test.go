package example

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// TestSimpleConvert 测试基本的视频转换功能
func TestSimpleConvert(t *testing.T) {
	// 输入和输出文件路径
	inputFile := "./sample_data/in1.mp4"
	outputFile := "./sample_data/simple_convert_test_output.mp4"

	// 确保输出目录存在
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("创建输出目录失败: %v", err)
	}

	// 记录开始时间
	startTime := time.Now()

	// 执行转换
	err := ffmpeg.Input(inputFile).
		Output(outputFile, ffmpeg.KwArgs{
			"vcodec": "libx264",
			"preset": "medium",
			"crf":    23,
			"acodec": "aac",
			"b:a":    "128k",
		}).
		OverWriteOutput().
		Run()

	// 计算执行时间
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	// 检查输出文件是否存在
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("输出文件未创建: %v", err)
	}

	// 输出执行时间
	fmt.Printf("转换成功，耗时: %v\n", duration)

	// 清理输出文件
	// defer os.Remove(outputFile)
}