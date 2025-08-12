package example

import (
	"fmt"
	"log"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func main() {
	fmt.Println("=== FFmpeg日志系统测试 ===")
	
	inputFile := "./sample_data/in1.mp4"
	outputFile := "./sample_data/logging_output.mp4"
	
	// 启用全局日志
	ffmpeg.SetGlobalOptions(ffmpeg.KwArgs{
		"loglevel": "verbose",
	})
	
	// 执行转换并查看详细日志
	fmt.Println("\n执行视频转换（详细日志）:")
	err := ffmpeg.Input(inputFile).
		Output(outputFile, ffmpeg.KwArgs{
			"vcodec": "libx264",
			"preset": "medium",
			"crf":    23,
		}).
		OverWriteOutput().
		Run()
	
	if err != nil {
		log.Printf("转换失败: %v", err)
	} else {
		fmt.Println("转换成功完成")
	}
	
	// 测试静默模式
	fmt.Println("\n执行静默模式转换:")
	ffmpeg.SetGlobalOptions(ffmpeg.KwArgs{
		"hide_banner": "",
		"loglevel":    "quiet",
	})
	
	err = ffmpeg.Input(inputFile).
		Output(outputFile+"_quiet.mp4").
		OverWriteOutput().
		Run()
	
	if err != nil {
		log.Printf("静默模式转换失败: %v", err)
	} else {
		fmt.Println("静默模式转换成功完成")
	}
	
	fmt.Println("\n=== 日志系统测试完成 ===")
}
