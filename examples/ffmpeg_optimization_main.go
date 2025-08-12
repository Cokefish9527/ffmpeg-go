package example

import (
	"fmt"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func main() {
	fmt.Println("=== FFmpeg性能优化测试 ===")
	
	// 测试文件路径
	inputFile := "./sample_data/in1.mp4"
	outputFile1 := "./sample_data/optimized_output1.mp4"
	outputFile2 := "./sample_data/optimized_output2.mp4"
	
	// 测试1: 基础转换
	fmt.Println("\n1. 基础转换:")
	start := time.Now()
	err := ffmpeg.Input(inputFile).
		Output(outputFile1).
		OverWriteOutput().
		Run()
	duration1 := time.Since(start)
	
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
	} else {
		fmt.Printf("   耗时: %v\n", duration1)
	}
	
	// 测试2: 优化参数转换
	fmt.Println("\n2. 优化参数转换:")
	start = time.Now()
	err = ffmpeg.Input(inputFile).
		Output(outputFile2, ffmpeg.KwArgs{
			"vcodec": "libx264",
			"preset": "fast",
			"crf":    23,
			"acodec": "aac",
			"threads": 4,
		}).
		OverWriteOutput().
		Run()
	duration2 := time.Since(start)
	
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
	} else {
		fmt.Printf("   耗时: %v\n", duration2)
		fmt.Printf("   性能提升: %v\n", duration1-duration2)
	}
	
	fmt.Println("\n=== 性能优化测试完成 ===")
}

func testEncoders() {
	fmt.Println("\n=== 测试不同编码器 ===")
	
	// 测试libx264编码器
	testEncoder("libx264", "ultrafast")
	
	// 如果有NVENC，测试NVENC编码器
	if isEncoderAvailable("h264_nvenc") {
		testEncoder("h264_nvenc", "fast")
	}
	
	// 如果有QSV，测试QSV编码器
	if isEncoderAvailable("h264_qsv") {
		testEncoder("h264_qsv", "fast")
	}
	
	// 如果有AMF，测试AMF编码器
	if isEncoderAvailable("h264_amf") {
		testEncoder("h264_amf", "fast")
	}
}

func testEncoder(encoder, preset string) {
	fmt.Printf("测试编码器: %s, 预设: %s\n", encoder, preset)
	
	start := time.Now()
	
	// 创建一个简单的测试命令
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "testsrc=duration=5:size=1280x720:rate=30",
		"-c:v", encoder, "-preset", preset, "-t", "5", "-y", fmt.Sprintf("test_%s.mp4", encoder))
	
	err := cmd.Run()
	if err != nil {
		fmt.Printf("编码器 %s 测试失败: %v\n", encoder, err)
		return
	}
	
	duration := time.Since(start)
	fmt.Printf("编码器 %s 测试完成，耗时: %v\n", encoder, duration)
	
	// 清理测试文件
	os.Remove(fmt.Sprintf("test_%s.mp4", encoder))
}

func testPresets() {
	fmt.Println("\n=== 测试不同预设 ===")
	
	presets := []string{"ultrafast", "superfast", "veryfast", "faster", "fast", "medium"}
	
	for _, preset := range presets {
		fmt.Printf("测试预设: %s\n", preset)
		
		start := time.Now()
		
		// 创建一个简单的测试命令
		cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "testsrc=duration=5:size=1280x720:rate=30",
			"-c:v", "libx264", "-preset", preset, "-t", "5", "-y", fmt.Sprintf("test_%s.mp4", preset))
		
		err := cmd.Run()
		if err != nil {
			fmt.Printf("预设 %s 测试失败: %v\n", preset, err)
			continue
		}
		
		duration := time.Since(start)
		fmt.Printf("预设 %s 测试完成，耗时: %v\n", preset, duration)
		
		// 清理测试文件
		os.Remove(fmt.Sprintf("test_%s.mp4", preset))
	}
}

func testHardwareEncoderDetection() {
	fmt.Println("\n=== 测试硬件编码器检测 ===")
	
	encoders := detectHardwareEncoders()
	
	fmt.Println("检测到的硬件编码器:")
	for encoder, available := range encoders {
		if available {
			fmt.Printf("  %s: 可用\n", encoder)
		} else {
			fmt.Printf("  %s: 不可用\n", encoder)
		}
	}
}

// 检测可用的硬件编码器
func detectHardwareEncoders() map[string]bool {
	encoders := make(map[string]bool)
	
	// 检测NVIDIA NVENC
	encoders["h264_nvenc"] = isEncoderAvailable("h264_nvenc")
	
	// 检测Intel Quick Sync
	encoders["h264_qsv"] = isEncoderAvailable("h264_qsv")
	
	// 检测AMD VCE
	encoders["h264_amf"] = isEncoderAvailable("h264_amf")
	
	return encoders
}

// 检查编码器是否可用
func isEncoderAvailable(encoder string) bool {
	cmd := exec.Command("ffmpeg", "-h", fmt.Sprintf("encoder=%s", encoder))
	return cmd.Run() == nil
}