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

func ConvertMP4ToTS() {
	// 指定要处理的目录
	inputDir := "./temp"
	
	// 检查目录是否存在
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		fmt.Printf("目录 %s 不存在\n", inputDir)
		return
	}
	
	// 创建任务队列
	taskQueue := queue.NewInMemoryTaskQueue()
	
	// 创建工作池
	workerPool := service.NewWorkerPool(3, taskQueue)
	
	// 启动工作池
	workerPool.Start()
	
	// 用于收集所有任务ID，以便后续查询状态
	var taskIDs []string
	
	// 遍历目录中的所有mp4文件
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 只处理mp4文件
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".mp4" {
			// 生成任务ID
			taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
			taskIDs = append(taskIDs, taskID)
			
			fmt.Printf("准备转换文件: %s\n", path)
			
			// 创建任务
			task := &queue.Task{
				ID:   taskID,
				Spec: map[string]interface{}{
					"source":   path,
					"taskType": "convertMP4ToTS",
				},
				Status:   "pending",
				Created:  time.Now(),
				Progress: 0.0,
			}
			
			// 将任务添加到队列
			if err := taskQueue.Push(task); err != nil {
				fmt.Printf("添加任务失败: %v\n", err)
				return err
			}
		}
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("遍历目录时出错: %v\n", err)
		return
	}
	
	if len(taskIDs) == 0 {
		fmt.Println("未找到任何MP4文件")
		return
	}
	
	fmt.Printf("已提交 %d 个转换任务\n", len(taskIDs))
	
	// 等待所有任务完成
	fmt.Println("等待任务完成...")
	
	// 轮询任务状态直到所有任务完成
	for {
		allCompleted := true
		completedCount := 0
		
		for _, taskID := range taskIDs {
			task, err := taskQueue.Get(taskID)
			if err != nil {
				fmt.Printf("获取任务 %s 状态时出错: %v\n", taskID, err)
				continue
			}
			
			if task.Status == "completed" {
				completedCount++
			} else if task.Status != "pending" && task.Status != "processing" {
				// 任务失败或其他终止状态
				completedCount++
				fmt.Printf("任务 %s 失败: %s\n", taskID, task.Error)
			} else {
				// 任务仍在处理中
				allCompleted = false
			}
		}
		
		fmt.Printf("进度: %d/%d 任务完成\n", completedCount, len(taskIDs))
		
		if allCompleted {
			break
		}
		
		// 等待一段时间再检查
		time.Sleep(1 * time.Second)
	}
	
	// 停止工作池
	workerPool.Stop()
	
	fmt.Println("所有转换任务已完成")
	
	// 显示最终结果
	for _, taskID := range taskIDs {
		task, err := taskQueue.Get(taskID)
		if err != nil {
			fmt.Printf("获取任务 %s 状态时出错: %v\n", taskID, err)
			continue
		}
		
		if task.Status == "completed" {
			fmt.Printf("✓ 任务 %s 完成: %s\n", taskID, task.Result)
		} else {
			fmt.Printf("✗ 任务 %s 失败: %s\n", taskID, task.Error)
		}
	}
}