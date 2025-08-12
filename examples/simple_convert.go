package example

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/u2takey/ffmpeg-go/service"
	"github.com/u2takey/ffmpeg-go/queue"
)

func main() {
	// 指定要处理的目录
	inputDir := "./temp"
	
	// 检查目录是否存在
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		fmt.Printf("目录 %s 不存在\n", inputDir)
		return
	}
	
	// 创建素材预处理服务
	preprocessor := service.NewMaterialPreprocessorService()
	
	// 用于统计总转换时间和文件数量
	var totalDuration time.Duration
	convertedFiles := 0
	
	fmt.Println("开始转换MP4文件为TS格式...")
	fmt.Println(strings.Repeat("=", 50))
	
	// 遍历目录中的所有mp4文件
	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 只处理mp4文件
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".mp4" {
			// 获取源文件信息
			srcFileInfo, err := os.Stat(path)
			if err != nil {
				fmt.Printf("无法获取源文件信息 %s: %v\n", path, err)
				return nil
			}
			
			fmt.Printf("准备转换文件: %s\n", path)
			fmt.Printf("  源文件大小: %.2f MB\n", float64(srcFileInfo.Size())/(1024*1024))
			
			// 生成输出文件路径 (TS格式)
			ext := filepath.Ext(path)
			outputFile := path[0:len(path)-len(ext)] + ".ts"
			
			// 创建任务对象
			task := &queue.Task{
				ID:       "test-task",
				Spec: map[string]interface{}{
					"source":   path,
					"taskType": "materialPreprocess",
				},
				Status:   "pending",
				Progress: 0.0,
			}
			
			// 记录开始时间
			startTime := time.Now()
			
			// 处理任务
			err = preprocessor.Process(task)
			if err != nil {
				fmt.Printf("转换文件 %s 失败: %v\n", path, err)
				return nil
			}
			
			// 记录结束时间
			endTime := time.Now()
			duration := endTime.Sub(startTime)
			totalDuration += duration
			
			// 获取目标文件信息
			dstFileInfo, err := os.Stat(outputFile)
			if err != nil {
				fmt.Printf("无法获取目标文件信息 %s: %v\n", outputFile, err)
				return nil
			}
			
			// 计算文件大小变化
			srcSize := float64(srcFileInfo.Size())
			dstSize := float64(dstFileInfo.Size())
			sizeChange := (dstSize - srcSize) / srcSize * 100
			
			fmt.Printf("✓ 成功转换文件: %s -> %s\n", path, outputFile)
			fmt.Printf("  转换耗时: %v\n", duration)
			fmt.Printf("  目标文件大小: %.2f MB (%+.2f%%)\n", dstSize/(1024*1024), sizeChange)
			fmt.Println()
			
			convertedFiles++
		}
		
		return nil
	})
	
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("转换完成总结:\n")
	fmt.Printf("  成功转换文件数: %d\n", convertedFiles)
	fmt.Printf("  总转换耗时: %v\n", totalDuration)
	if convertedFiles > 0 {
		fmt.Printf("  平均转换耗时: %v\n", totalDuration/time.Duration(convertedFiles))
	}
	fmt.Println("所有转换任务已完成")
}