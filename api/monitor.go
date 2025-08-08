package api

import (
	"fmt"
	"net/http"
	"runtime"
	"time"
	"os"
	"path/filepath"
	"io/ioutil"

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
// @Description 任务重试请求参数
type TaskRetryRequest struct {
	// 任务ID
	TaskID string `json:"taskId" binding:"required"`
}

// TaskCancelRequest 任务取消请求
// @Description 任务取消请求参数
type TaskCancelRequest struct {
	// 任务ID
	TaskID string `json:"taskId" binding:"required"`
}

// TaskDiscardRequest 任务丢弃请求
// @Description 任务丢弃请求参数
type TaskDiscardRequest struct {
	// 任务ID
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
// @Description 系统统计信息
type SystemStats struct {
	// 时间戳
	Timestamp     time.Time `json:"timestamp"`
	// CPU使用率
	CPUUsage      float64   `json:"cpuUsage"`
	// 内存使用率
	MemoryUsage   float64   `json:"memoryUsage"`
	// 总内存
	MemoryTotal   uint64    `json:"memoryTotal"`
	// 已使用内存
	MemoryUsed    uint64    `json:"memoryUsed"`
	// 磁盘使用率
	DiskUsage     float64   `json:"diskUsage"`
	// 总磁盘空间
	DiskTotal     uint64    `json:"diskTotal"`
	// 已使用磁盘空间
	DiskUsed      uint64    `json:"diskUsed"`
	// Goroutines数量
	Goroutines    int       `json:"goroutines"`
	// 工作线程总数
	WorkerCount   int       `json:"workerCount"`
	// 活跃工作线程数
	ActiveWorkers int       `json:"activeWorkers"`
	// 任务队列大小
	TaskQueueSize int       `json:"taskQueueSize"`
}

// TaskStats 任务统计信息
// @Description 任务统计信息
type TaskStats struct {
	// 总任务数
	TotalTasks     int `json:"totalTasks"`
	// 待处理任务数
	PendingTasks   int `json:"pendingTasks"`
	// 处理中任务数
	ProcessingTasks int `json:"processingTasks"`
	// 已完成任务数
	CompletedTasks int `json:"completedTasks"`
	// 失败任务数
	FailedTasks    int `json:"failedTasks"`
}

// GetSystemStats 获取系统统计信息
// @Summary 获取系统统计信息
// @Description 获取系统资源使用情况统计信息，包括CPU、内存、磁盘等
// @Tags monitor
// @Produce json
// @Success 200 {object} SystemStats "系统统计信息"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/stats [get]
func (m *MonitorAPI) GetSystemStats(c *gin.Context) {
	utils.Debug("收到系统统计信息请求", map[string]string{"clientIP": c.ClientIP()})
	
	// 获取CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		utils.Error("获取CPU使用率失败", map[string]string{"error": err.Error()})
		cpuPercent = []float64{0}
	}

	// 确保CPU使用率在合理范围内
	var totalUsage float64
	for _, usage := range cpuPercent {
		totalUsage += usage
	}
	cpuCount := runtime.NumCPU()
	if totalUsage > float64(cpuCount) * 100 {
		totalUsage = float64(cpuCount) * 100
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
	
	// 获取Worker统计信息
	workerCount := m.workerPool.GetWorkerCount()
	activeWorkers := m.workerPool.GetActiveWorkerCount()
	
	stats := &SystemStats{
		Timestamp:     time.Now(),
		CPUUsage:      totalUsage,
		MemoryUsage:   memInfo.UsedPercent,
		MemoryTotal:   memInfo.Total,
		MemoryUsed:    memInfo.Used,
		DiskUsage:     diskInfo.UsedPercent,
		DiskTotal:     diskInfo.Total,
		DiskUsed:      diskInfo.Used,
		Goroutines:    runtime.NumGoroutine(),
		WorkerCount:   workerCount,
		ActiveWorkers: activeWorkers,
		TaskQueueSize: len(tasks),
	}
	
	utils.Info("系统统计信息获取成功", nil)
	c.JSON(http.StatusOK, stats)
}

// GetTaskStats 获取任务统计信息
// @Summary 获取任务统计信息
// @Description 获取任务统计信息，包括各种状态的任务数量
// @Tags monitor
// @Produce json
// @Success 200 {object} TaskStats "任务统计信息"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks/stats [get]
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
// @Summary 获取任务列表
// @Description 获取所有任务列表，支持按状态和优先级筛选
// @Tags monitor
// @Produce json
// @Param status query string false "任务状态筛选"
// @Param priority query string false "任务优先级筛选"
// @Success 200 {array} queue.Task "任务列表"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks [get]
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
// @Summary 获取任务详情
// @Description 根据任务ID获取任务详细信息
// @Tags monitor
// @Produce json
// @Param taskId path string true "任务ID"
// @Success 200 {object} queue.Task "任务详情"
// @Failure 404 {object} map[string]string "任务未找到"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks/{taskId} [get]
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
// @Summary 获取Worker统计信息
// @Description 获取Worker池的统计信息
// @Tags monitor
// @Produce json
// @Success 200 {object} map[string]interface{} "Worker统计信息"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/workers [get]
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
// @Summary 重试失败的任务
// @Description 重试一个失败的任务
// @Tags monitor
// @Accept json
// @Produce json
// @Param request body TaskRetryRequest true "任务重试请求"
// @Success 200 {object} map[string]string "重试成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "任务未找到"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks/retry [post]
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
// @Summary 取消任务
// @Description 取消一个待处理或处理中的任务
// @Tags monitor
// @Accept json
// @Produce json
// @Param request body TaskCancelRequest true "任务取消请求"
// @Success 200 {object} map[string]string "取消成功"
// @Failure 400 {object} map[string]string "请求参数错误或任务状态不正确"
// @Failure 404 {object} map[string]string "任务未找到"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks/cancel [post]
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
// @Summary 丢弃任务
// @Description 丢弃一个已完成或失败的任务
// @Tags monitor
// @Accept json
// @Produce json
// @Param request body TaskDiscardRequest true "任务丢弃请求"
// @Success 200 {object} map[string]string "丢弃成功"
// @Failure 400 {object} map[string]string "请求参数错误或任务状态不正确"
// @Failure 404 {object} map[string]string "任务未找到"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks/discard [post]
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

// GetTaskExecutions 获取任务的所有执行历史
// @Summary 获取任务执行历史
// @Description 获取指定任务的所有执行历史记录
// @Tags monitor
// @Produce json
// @Param taskId path string true "任务ID"
// @Success 200 {array} queue.TaskExecution "任务执行历史记录列表"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks/{taskId}/executions [get]
func (m *MonitorAPI) GetTaskExecutions(c *gin.Context) {
    taskId := c.Param("taskId")
    utils.Debug("收到任务执行历史请求", map[string]string{
        "taskId":   taskId,
        "clientIP": c.ClientIP(),
    })
    
    // 获取任务执行历史
    executions, err := m.taskQueue.GetTaskExecutions(taskId)
    if err != nil {
        utils.Error("获取任务执行历史失败", map[string]string{
            "taskId": taskId,
            "error":  err.Error(),
        })
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to get task executions",
        })
        return
    }
    
    utils.Info("任务执行历史获取成功", map[string]string{"taskId": taskId})
    c.JSON(http.StatusOK, executions)
}

// GetTaskLog 获取任务日志
// @Summary 获取任务日志
// @Description 获取指定任务的日志内容
// @Tags monitor
// @Produce json
// @Param taskId path string true "任务ID"
// @Success 200 {object} map[string]string "任务日志内容"
// @Failure 404 {object} map[string]string "任务日志未找到"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /monitor/tasks/{taskId}/log [get]
func (m *MonitorAPI) GetTaskLog(c *gin.Context) {
	utils.Debug("收到任务日志请求", map[string]string{"clientIP": c.ClientIP()})

	taskID := c.Param("taskId")
	if taskID == "" {
		utils.Warn("任务ID不能为空", nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}

	// 构建日志文件路径
	logDir := "./log/tasks"
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", taskID))

	// 检查日志文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		utils.Warn("任务日志文件不存在", map[string]string{"taskId": taskID})
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task log not found",
		})
		return
	}

	// 读取日志文件内容
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		utils.Error("读取任务日志文件失败", map[string]string{
			"taskId": taskID,
			"error":  err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read task log",
		})
		return
	}

	utils.Info("任务日志获取成功", map[string]string{"taskId": taskID})
	c.JSON(http.StatusOK, gin.H{
		"taskId": taskID,
		"log":    string(content),
	})
}