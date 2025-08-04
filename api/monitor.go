package api

import (
	"fmt"
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

// MonitorAPI 监控API结构体
type MonitorAPI struct {
	taskQueue  queue.TaskQueue
	workerPool *service.WorkerPool
}

// TaskRetryRequest 任务重试请求
type TaskRetryRequest struct {
	TaskID string `json:"taskId" binding:"required"`
}

// TaskCancelRequest 任务取消请求
type TaskCancelRequest struct {
	TaskID string `json:"taskId" binding:"required"`
}

// TaskDiscardRequest 任务丢弃请求
type TaskDiscardRequest struct {
	TaskID string `json:"taskId" binding:"required"`
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

// RetryTask 重试失败的任务
func (m *MonitorAPI) RetryTask(c *gin.Context) {
	utils.Debug("收到任务重试请求", map[string]string{"clientIP": c.ClientIP()})
	
	var req TaskRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warn("任务重试请求格式错误", map[string]string{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	utils.Info("开始处理任务重试请求", map[string]string{"taskId": req.TaskID})
	
	// 获取任务
	task, err := m.taskQueue.Get(req.TaskID)
	if err != nil {
		utils.Error("获取任务失败", map[string]string{
			"taskId": req.TaskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task",
		})
		return
	}
	
	if task == nil {
		utils.Warn("任务不存在", map[string]string{"taskId": req.TaskID})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
		})
		return
	}
	
	utils.Info("获取到需要重试的任务", map[string]string{
		"taskId": req.TaskID,
		"status": task.Status,
		"executionCount": fmt.Sprintf("%d", task.ExecutionCount),
	})
	
	// 只有失败的任务才能重试
	if task.Status != "failed" {
		utils.Warn("任务状态不正确，无法重试", map[string]string{
			"taskId": req.TaskID,
			"status": task.Status,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only failed tasks can be retried",
		})
		return
	}
	
	// 重置任务状态
	task.Status = "pending"
	task.Error = ""
	task.Started = time.Time{}
	task.Finished = time.Time{}
	task.Progress = 0.0
	
	// 更新任务
	err = m.taskQueue.Update(task)
	if err != nil {
		utils.Error("更新任务失败", map[string]string{
			"taskId": req.TaskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update task",
		})
		return
	}
	
	// 将任务重新推入队列等待处理
	err = m.taskQueue.Push(task)
	if err != nil {
		utils.Error("重新推入任务队列失败", map[string]string{
			"taskId": req.TaskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to push task to queue",
		})
		return
	}
	
	utils.Info("任务重试成功", map[string]string{
		"taskId": req.TaskID,
		"executionCount": fmt.Sprintf("%d", task.ExecutionCount),
	})
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Task retry successfully",
		"taskId":  req.TaskID,
	})
}

// CancelTask 取消任务
func (m *MonitorAPI) CancelTask(c *gin.Context) {
	utils.Debug("收到任务取消请求", map[string]string{"clientIP": c.ClientIP()})
	
	var req TaskCancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warn("任务取消请求格式错误", map[string]string{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	// 获取任务
	task, err := m.taskQueue.Get(req.TaskID)
	if err != nil {
		utils.Error("获取任务失败", map[string]string{
			"taskId": req.TaskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task",
		})
		return
	}
	
	if task == nil {
		utils.Warn("任务不存在", map[string]string{"taskId": req.TaskID})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
		})
		return
	}
	
	// 只有待处理和处理中的任务可以取消
	if task.Status != "pending" && task.Status != "processing" {
		utils.Warn("任务状态不正确，无法取消", map[string]string{
			"taskId": req.TaskID,
			"status": task.Status,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only pending or processing tasks can be cancelled",
		})
		return
	}
	
	// 更新任务状态为已取消
	task.Status = "cancelled"
	task.Finished = time.Now()
	
	// 更新任务
	err = m.taskQueue.Update(task)
	if err != nil {
		utils.Error("更新任务失败", map[string]string{
			"taskId": req.TaskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update task",
		})
		return
	}
	
	utils.Info("任务取消成功", map[string]string{"taskId": req.TaskID})
	c.JSON(http.StatusOK, gin.H{
		"message": "Task cancelled successfully",
		"taskId":  req.TaskID,
	})
}

// DiscardTask 丢弃任务
func (m *MonitorAPI) DiscardTask(c *gin.Context) {
	utils.Debug("收到任务丢弃请求", map[string]string{"clientIP": c.ClientIP()})
	
	var req TaskDiscardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warn("任务丢弃请求格式错误", map[string]string{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	utils.Debug("解析任务丢弃请求", map[string]string{"taskId": req.TaskID})
	
	// 获取任务
	task, err := m.taskQueue.Get(req.TaskID)
	if err != nil {
		utils.Error("获取任务失败", map[string]string{
			"taskId": req.TaskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task",
		})
		return
	}
	
	if task == nil {
		utils.Warn("任务不存在", map[string]string{"taskId": req.TaskID})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
		})
		return
	}
	
	utils.Debug("获取到任务", map[string]string{"taskId": req.TaskID, "status": task.Status})
	
	// 只有失败或已完成的任务才能被丢弃
	if task.Status != "failed" && task.Status != "completed" {
		utils.Warn("任务状态不正确，无法丢弃", map[string]string{
			"taskId": req.TaskID,
			"status": task.Status,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only failed or completed tasks can be discarded",
		})
		return
	}
	
	// 从队列中移除任务
	// 注意：当前的内存队列实现不支持直接删除任务
	// 我们将任务状态设置为"discarded"来表示已丢弃
	task.Status = "discarded"
	task.Finished = time.Now()
	
	err = m.taskQueue.Update(task)
	if err != nil {
		utils.Error("更新任务状态失败", map[string]string{
			"taskId": req.TaskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update task",
		})
		return
	}
	
	utils.Info("任务丢弃成功", map[string]string{"taskId": req.TaskID})
	c.JSON(http.StatusOK, gin.H{
		"message": "Task discarded successfully",
		"taskId":  req.TaskID,
	})
}
