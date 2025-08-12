package example

import (
	"fmt"
	"sync"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// 视频处理任务结构
type VideoTask struct {
	InputFile  string
	OutputFile string
	Preset     string
}

// 执行单个视频处理任务
func processVideoTask(task VideoTask, wg *sync.WaitGroup) {
	defer wg.Done()
	
	start := time.Now()
	fmt.Printf("开始处理: %s -> %s (预设: %s)\n", task.InputFile, task.OutputFile, task.Preset)
	
	err := ffmpeg.Input(task.InputFile).
		Output(task.OutputFile, ffmpeg.KwArgs{
			"vcodec": "libx264",
			"preset": task.Preset,
			"crf":    23,
		}).
		OverWriteOutput().
		Run()
	
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("处理失败 %s: %v\n", task.InputFile, err)
	} else {
		fmt.Printf("处理完成 %s -> %s，耗时: %v (预设: %s)\n", 
			task.InputFile, task.OutputFile, duration, task.Preset)
	}
}

func main() {
	fmt.Println("=== 并行视频处理测试 ===")
	
	// 定义处理任务
	tasks := []VideoTask{
		{"./sample_data/in1.mp4", "./sample_data/parallel_output1.mp4", "ultrafast"},
		{"./sample_data/in2.mp4", "./sample_data/parallel_output2.mp4", "fast"},
		{"./sample_data/in3.mp4", "./sample_data/parallel_output3.mp4", "medium"},
	}
	
	var wg sync.WaitGroup
	
	// 并行处理所有任务
	startTime := time.Now()
	for _, task := range tasks {
		wg.Add(1)
		go processVideoTask(task, &wg)
	}
	
	// 等待所有任务完成
	wg.Wait()
	
	totalDuration := time.Since(startTime)
	fmt.Printf("\n所有任务完成，总耗时: %v\n", totalDuration)
	fmt.Println("=== 并行处理测试完成 ===")
}