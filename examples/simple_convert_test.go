package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/u2takey/ffmpeg-go/service"
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
	
	// 遍历目录中的所有mp4文件
	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 只处理mp4文件
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".mp4" {
			fmt.Printf("准备转换文件: %s\n", path)
			
			// 生成输出文件路径 (TS格式)
			ext := filepath.Ext(path)
			outputFile := path[0:len(path)-len(ext)] + ".ts"
			
			// 创建任务对象
			task := &service.Task{
				ID:       "test-task",
				Spec: map[string]interface{}{
					"source":   path,
					"taskType": "materialPreprocess",
				},
				Status:   "pending",
				Progress: 0.0,
			}
			
			// 处理任务
			err := preprocessor.Process(task)
			if err != nil {
				fmt.Printf("转换文件 %s 失败: %v\n", path, err)
				return nil
			}
			
			fmt.Printf("✓ 成功转换文件: %s -> %s\n", path, outputFile)
		}
		
		return nil
	})
	
	fmt.Println("所有转换任务已完成")
}