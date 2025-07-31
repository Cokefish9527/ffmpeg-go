package main

import (
	"log"
)

func main() {
	// 创建ffmpeg验证实例
	validation := NewFfmpegValidation()
	
	// 运行验证
	validation.RunValidation()
	
	log.Println("验证程序执行完毕")
}