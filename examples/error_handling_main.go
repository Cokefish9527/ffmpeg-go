package example

import (
	"fmt"
	"log"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func ErrorHandlingMain() {
	// 正确的文件路径
	correctFile := "./sample_data/in1.mp4"
	
	// 错误的文件路径
	wrongFile := "./sample_data/nonexistent.mp4"
	
	fmt.Println("=== 测试FFmpeg错误处理 ===")
	
	// 测试1: 尝试处理不存在的文件
	fmt.Println("\n1. 测试处理不存在的文件:")
	err := ffmpeg.Input(wrongFile).
		Output("./sample_data/error_test_output.mp4").
		Run()
	
	if err != nil {
		fmt.Printf("   捕获到错误: %v\n", err)
	} else {
		fmt.Println("   意外: 没有发生错误")
	}
	
	// 测试2: 尝试处理存在的文件
	fmt.Println("\n2. 测试处理存在的文件:")
	err = ffmpeg.Input(correctFile).
		Output("./sample_data/error_test_output.mp4").
		OverWriteOutput().
		Run()
	
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
	} else {
		fmt.Println("   成功处理文件")
	}
	
	// 测试3: 尝试使用不支持的编解码器
	fmt.Println("\n3. 测试使用不支持的编解码器:")
	err = ffmpeg.Input(correctFile).
		Output("./sample_data/error_test_output.mp4", ffmpeg.KwArgs{"vcodec": "unsupported_codec"}).
		OverWriteOutput().
		Run()
	
	if err != nil {
		fmt.Printf("   捕获到错误: %v\n", err)
		log.Printf("详细错误信息: %v", err)
	} else {
		fmt.Println("   意外: 没有发生错误")
	}
	
	fmt.Println("\n=== 错误处理测试完成 ===")
}