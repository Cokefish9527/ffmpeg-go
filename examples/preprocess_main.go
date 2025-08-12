package example

import (
	"fmt"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// 预处理视频：调整分辨率和帧率
func preprocessVideo(inputFile, outputFile string) error {
	fmt.Printf("预处理视频: %s -> %s\n", inputFile, outputFile)
	
	start := time.Now()
	err := ffmpeg.Input(inputFile).
		Output(outputFile, ffmpeg.KwArgs{
			"vf":       "scale=1280:720", // 调整分辨率到720p
			"r":        30,               // 设置帧率为30fps
			"vcodec":   "libx264",
			"preset":   "fast",
			"crf":      23,
			"acodec":   "aac",
			"ar":       44100,            // 音频采样率
			"ac":       2,                // 双声道
		}).
		OverWriteOutput().
		Run()
	
	duration := time.Since(start)
	
	if err != nil {
		return fmt.Errorf("预处理失败: %w", err)
	}
	
	fmt.Printf("预处理完成，耗时: %v\n", duration)
	return nil
}

// 视频分析函数
func analyzeVideo(inputFile string) error {
	fmt.Printf("分析视频: %s\n", inputFile)
	
	// 获取视频信息
	probeData, err := ffmpeg.Probe(inputFile)
	if err != nil {
		return fmt.Errorf("视频分析失败: %w", err)
	}
	
	fmt.Printf("视频信息: %.100s...\n", probeData)
	return nil
}

func main() {
	fmt.Println("=== 视频预处理测试 ===")
	
	inputFile := "./sample_data/in1.mp4"
	preprocessedFile := "./sample_data/preprocessed_output.mp4"
	
	// 分析原始视频
	fmt.Println("\n1. 分析原始视频:")
	if err := analyzeVideo(inputFile); err != nil {
		fmt.Printf("   错误: %v\n", err)
	}
	
	// 预处理视频
	fmt.Println("\n2. 预处理视频:")
	if err := preprocessVideo(inputFile, preprocessedFile); err != nil {
		fmt.Printf("   错误: %v\n", err)
	} else {
		// 分析预处理后的视频
		fmt.Println("\n3. 分析预处理后的视频:")
		if err := analyzeVideo(preprocessedFile); err != nil {
			fmt.Printf("   错误: %v\n", err)
		}
	}
	
	fmt.Println("\n=== 视频预处理测试完成 ===")
}