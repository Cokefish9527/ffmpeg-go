package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
	"github.com/u2takey/ffmpeg-go/utils"
)

// MonitorAPI 监控API结构
type MonitorAPI struct {
	taskQueue  queue.TaskQueue
	workerPool *service.WorkerPool
}

// NewMonitorAPI 创建新的监控API实例
func NewMonitorAPI(taskQueue queue.TaskQueue, workerPool *service.WorkerPool) *MonitorAPI {
	return &MonitorAPI{
		taskQueue:  taskQueue,
		workerPool: workerPool,
	}
}

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

// GetSystemStats 获取系统统计信息
func (m *MonitorAPI) GetSystemStats(c *gin.Context) {
	utils.Debug("收到系统统计信息请求", map[string]string{"clientIP": c.ClientIP()})
	
	// 获取CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		utils.Error("获取CPU使用率失败", map[string]string{"error": err.Error()})
		cpuPercent = []float64{0}
	}
	
	// 获取内存信息
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		utils.Error("获取内存信息失败", map[string]string{"error": err.Error()})
		memInfo = &mem.VirtualMemoryStat{}
	}
	
	// 获取磁盘信息
	diskInfo, err := disk.Usage("/")
	if err != nil {
		utils.Error("获取磁盘信息失败", map[string]string{"error": err.Error()})
		diskInfo = &disk.UsageStat{}
	}
	
	// 获取任务队列大小
	tasks, err := m.taskQueue.List()
	if err != nil {
		utils.Error("获取任务列表失败", map[string]string{"error": err.Error()})
		tasks = []*queue.Task{}
	}
	
	stats := &SystemStats{
		Timestamp:     time.Now(),
		CPUUsage:      cpuPercent[0],
		MemoryUsage:   memInfo.UsedPercent,
		MemoryTotal:   memInfo.Total,
		MemoryUsed:    memInfo.Used,
		DiskUsage:     diskInfo.UsedPercent,
		DiskTotal:     diskInfo.Total,
		DiskUsed:      diskInfo.Used,
		Goroutines:    runtime.NumGoroutine(),
		WorkerCount:   m.workerPool.GetWorkerCount(),
		TaskQueueSize: len(tasks),
	}
	
	utils.Info("系统统计信息获取成功", nil)
	c.JSON(http.StatusOK, stats)
}

// GetTaskStats 获取任务统计信息
func (m *MonitorAPI) GetTaskStats(c *gin.Context) {
	utils.Debug("收到任务统计信息请求", map[string]string{"clientIP": c.ClientIP()})
	
	// 获取所有任务
	tasks, err := m.taskQueue.List()
	if err != nil {
		utils.Error("获取任务列表失败", map[string]string{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task list",
		})
		return
	}
	
	// 统计各类任务数量
	stats := &TaskStats{
		TotalTasks: len(tasks),
	}
	
	for _, task := range tasks {
		switch task.Status {
		case "pending":
			stats.PendingTasks++
		case "processing":
			stats.ProcessingTasks++
		case "completed":
			stats.CompletedTasks++
		case "failed":
			stats.FailedTasks++
		}
	}
	
	utils.Info("任务统计信息获取成功", map[string]string{
		"total":      string(rune(stats.TotalTasks)),
		"pending":    string(rune(stats.PendingTasks)),
		"processing": string(rune(stats.ProcessingTasks)),
		"completed":  string(rune(stats.CompletedTasks)),
		"failed":     string(rune(stats.FailedTasks)),
	})
	
	c.JSON(http.StatusOK, stats)
}

// GetTasks 获取任务列表
func (m *MonitorAPI) GetTasks(c *gin.Context) {
	utils.Debug("收到任务列表请求", map[string]string{"clientIP": c.ClientIP()})
	
	// 获取所有任务
	tasks, err := m.taskQueue.List()
	if err != nil {
		utils.Error("获取任务列表失败", map[string]string{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task list",
		})
		return
	}
	
	// 支持状态筛选
	status := c.Query("status")
	if status != "" {
		var filteredTasks []*queue.Task
		for _, task := range tasks {
			if task.Status == status {
				filteredTasks = append(filteredTasks, task)
			}
		}
		tasks = filteredTasks
	}
	
	// 支持优先级筛选
	priority := c.Query("priority")
	if priority != "" {
		var filteredTasks []*queue.Task
		for _, task := range tasks {
			if string(rune(task.Priority+'0')) == priority {
				filteredTasks = append(filteredTasks, task)
			}
		}
		tasks = filteredTasks
	}
	
	utils.Info("任务列表获取成功", map[string]string{"taskCount": string(rune(len(tasks)))})
	c.JSON(http.StatusOK, tasks)
}

// GetTaskDetail 获取任务详情
func (m *MonitorAPI) GetTaskDetail(c *gin.Context) {
	taskID := c.Param("taskId")
	utils.Debug("收到任务详情请求", map[string]string{
		"taskId":   taskID,
		"clientIP": c.ClientIP(),
	})
	
	// 获取任务详情
	task, err := m.taskQueue.Get(taskID)
	if err != nil {
		utils.Error("获取任务详情失败", map[string]string{
			"taskId": taskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task detail",
		})
		return
	}
	
	if task == nil {
		utils.Warn("任务不存在", map[string]string{"taskId": taskID})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
		})
		return
	}
	
	utils.Info("任务详情获取成功", map[string]string{"taskId": taskID})
	c.JSON(http.StatusOK, task)
}

// GetWorkerStats 获取Worker统计信息
func (m *MonitorAPI) GetWorkerStats(c *gin.Context) {
	utils.Debug("收到Worker统计信息请求", map[string]string{"clientIP": c.ClientIP()})
	
	// 获取Worker池统计信息
	workerCount := m.workerPool.GetWorkerCount()
	
	stats := gin.H{
		"workerCount": workerCount,
		"status":      "running",
	}
	
	utils.Info("Worker统计信息获取成功", map[string]string{"workerCount": string(rune(workerCount))})
	c.JSON(http.StatusOK, stats)
}