package examples

import (
	"fmt"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func main() {
	fmt.Println("Testing ffmpeg-go with a simple example")
	
	// 测试一个简单的命令，获取视频信息
	probeData, err := ffmpeg.Probe("./sample_data/in1.mp4")
	if err != nil {
		fmt.Printf("Error probing video: %v\n", err)
		return
	}
	
	fmt.Printf("Probe successful. Sample of data: %.100s...\n", probeData)
	
	// 测试一个简单的转换命令
	err = ffmpeg.Input("./sample_data/in1.mp4").
		Output("./sample_data/output_test.mp4").
		OverWriteOutput().
		Run()
	
	if err != nil {
		fmt.Printf("Error running ffmpeg: %v\n", err)
		return
	}
	
	fmt.Println("Simple conversion test completed successfully!")
}