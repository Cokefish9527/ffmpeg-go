package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/u2takey/ffmpeg-go/utils"
)

func main() {
	fmt.Println("开始测试Goroutine池优化...")
	
	// 初始化日志系统
	utils.InitGlobalLogger()
	
	// 测试基本功能
	testBasicFunctionality()
	
	fmt.Println("Goroutine池测试完成")
}

func testBasicFunctionality() {
	fmt.Println("\n=== 测试基本功能 ===")
	
	// 创建Goroutine池
	pool := utils.NewGoroutinePool(
		utils.WithMinWorkers(2),
		utils.WithMaxWorkers(10),
		utils.WithTaskQueueSize(100),
	)
	
	// 启动池
	pool.Start()
	
	// 提交一些任务
	var wg sync.WaitGroup
	completed := 0
	var mutex sync.Mutex
	
	// 提交5个任务
	for i := 0; i < 5; i++ {
		wg.Add(1)
		taskID := i
		err := pool.SubmitFunc(func() error {
			defer wg.Done()
			// 模拟任务处理
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			mutex.Lock()
			completed++
			mutex.Unlock()
			utils.Info("任务完成", map[string]string{"taskId": fmt.Sprintf("%d", taskID)})
			return nil
		})
		
		if err != nil {
			fmt.Printf("提交任务%d失败: %v\n", i, err)
			wg.Done() // 如果提交失败，需要减少计数
		}
	}
	
	// 等待所有任务完成
	wg.Wait()
	
	// 获取统计信息
	stats := pool.GetStats()
	fmt.Printf("统计信息: %+v\n", stats)
	fmt.Printf("完成任务数: %d\n", completed)
	
	// 停止池
	pool.Stop()
}


