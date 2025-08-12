package example

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SystemStats 系统统计信息
type SystemStats struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpuUsage"`
	MemoryUsage   float64   `json:"memoryUsage"`
	MemoryTotal   uint64    `json:"memoryTotal"`
	MemoryUsed    uint64    `json:"memoryUsed"`
	DiskUsage     float64   `json:"diskUsage"`
	DiskTotal     uint64    `json:"diskTotal"`
	DiskUsed      uint64    `json:"diskUsed"`
	Goroutines    int       `json:"goroutines"`
	WorkerCount   int       `json:"workerCount"`
	TaskQueueSize int       `json:"taskQueueSize"`
}

// TaskStats 任务统计信息
type TaskStats struct {
	TotalTasks     int `json:"totalTasks"`
	PendingTasks   int `json:"pendingTasks"`
	ProcessingTasks int `json:"processingTasks"`
	CompletedTasks int `json:"completedTasks"`
	FailedTasks    int `json:"failedTasks"`
}

func main() {
	fmt.Println("开始测试监控API...")
	
	// 测试系统统计信息
	testSystemStats()
	
	// 测试任务统计信息
	testTaskStats()
	
	// 测试任务列表
	testTaskList()
	
	// 测试Worker统计信息
	testWorkerStats()
	
	fmt.Println("监控API测试完成")
}

func testSystemStats() {
	fmt.Println("\n=== 测试系统统计信息 ===")
	
	resp, err := http.Get("http://localhost:8082/api/v1/monitor/stats")
	if err != nil {
		fmt.Printf("请求系统统计信息失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("获取系统统计信息失败，状态码: %d\n", resp.StatusCode)
		return
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return
	}
	
	var stats SystemStats
	if err := json.Unmarshal(body, &stats); err != nil {
		fmt.Printf("解析响应体失败: %v\n", err)
		return
	}
	
	fmt.Printf("CPU使用率: %.2f%%\n", stats.CPUUsage)
	fmt.Printf("内存使用率: %.2f%%\n", stats.MemoryUsage)
	fmt.Printf("磁盘使用率: %.2f%%\n", stats.DiskUsage)
	fmt.Printf("Goroutines数量: %d\n", stats.Goroutines)
	fmt.Printf("Worker数量: %d\n", stats.WorkerCount)
	fmt.Printf("任务队列大小: %d\n", stats.TaskQueueSize)
}

func testTaskStats() {
	fmt.Println("\n=== 测试任务统计信息 ===")
	
	resp, err := http.Get("http://localhost:8082/api/v1/monitor/tasks/stats")
	if err != nil {
		fmt.Printf("请求任务统计信息失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("获取任务统计信息失败，状态码: %d\n", resp.StatusCode)
		return
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return
	}
	
	var stats TaskStats
	if err := json.Unmarshal(body, &stats); err != nil {
		fmt.Printf("解析响应体失败: %v\n", err)
		return
	}
	
	fmt.Printf("总任务数: %d\n", stats.TotalTasks)
	fmt.Printf("待处理任务: %d\n", stats.PendingTasks)
	fmt.Printf("处理中任务: %d\n", stats.ProcessingTasks)
	fmt.Printf("已完成任务: %d\n", stats.CompletedTasks)
	fmt.Printf("失败任务: %d\n", stats.FailedTasks)
}

func testTaskList() {
	fmt.Println("\n=== 测试任务列表 ===")
	
	resp, err := http.Get("http://localhost:8082/api/v1/monitor/tasks")
	if err != nil {
		fmt.Printf("请求任务列表失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("获取任务列表失败，状态码: %d\n", resp.StatusCode)
		return
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return
	}
	
	var tasks []interface{}
	if err := json.Unmarshal(body, &tasks); err != nil {
		fmt.Printf("解析响应体失败: %v\n", err)
		return
	}
	
	fmt.Printf("任务数量: %d\n", len(tasks))
	
	// 显示前几个任务的信息
	for i, task := range tasks {
		if i >= 3 {
			break
		}
		taskBytes, _ := json.Marshal(task)
		fmt.Printf("任务%d: %s\n", i+1, string(taskBytes))
	}
}

func testWorkerStats() {
	fmt.Println("\n=== 测试Worker统计信息 ===")
	
	resp, err := http.Get("http://localhost:8082/api/v1/monitor/workers")
	if err != nil {
		fmt.Printf("请求Worker统计信息失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("获取Worker统计信息失败，状态码: %d\n", resp.StatusCode)
		return
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return
	}
	
	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		fmt.Printf("解析响应体失败: %v\n", err)
		return
	}
	
	fmt.Printf("Worker统计信息: %v\n", stats)
}