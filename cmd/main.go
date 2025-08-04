package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
	"github.com/u2takey/ffmpeg-go/utils"
)

// TaskStatusResponse 任务状态响应
type TaskStatusResponse struct {
	TaskID    string             `json:"taskId"`
	Status    string             `json:"status"`
	Progress  float64            `json:"progress"`
	Message   string             `json:"message,omitempty"`
	Created   string             `json:"created,omitempty"`
	Started   string             `json:"started,omitempty"`
	Finished  string             `json:"finished,omitempty"`
	OutputURL string             `json:"outputUrl,omitempty"`
	Priority  queue.TaskPriority `json:"priority,omitempty"` // 添加优先级字段
}

var (
	taskQueue queue.TaskQueue // 使用queue包中的TaskQueue接口
	editorService service.VideoEditor
	workerPool *service.WorkerPool
	monitorAPI *api.MonitorAPI
)

func main() {
	// 初始化全局日志记录器
	utils.InitGlobalLogger()
	utils.Info("服务启动中", map[string]string{"phase": "initialization"})
	
	// 设置Gin运行模式
	gin.SetMode(gin.ReleaseMode)
	
	// 初始化任务队列 (使用queue包中的实现)
	taskQueue = queue.NewInMemoryTaskQueue()
	utils.Info("任务队列初始化完成", nil)
	
	// 从环境变量获取最大工作线程数，默认为0（使用CPU核心数）
	maxWorkers := 0
	if maxWorkersStr := os.Getenv("MAX_WORKERS"); maxWorkersStr != "" {
		if num, err := strconv.Atoi(maxWorkersStr); err == nil {
			maxWorkers = num
		}
	}
	utils.Info("工作线程数配置", map[string]string{"maxWorkers": strconv.Itoa(maxWorkers)})
	
	// 创建工作池
	workerPool = service.NewWorkerPool(maxWorkers, taskQueue)
	utils.Info("工作池创建完成", nil)
	
	// 启动工作池
	workerPool.Start()
	utils.Info("工作池启动完成", nil)
	
	// 创建监控API
	monitorAPI = api.NewMonitorAPI(taskQueue, workerPool)
	utils.Info("监控API创建完成", nil)
	
	// 启动一个goroutine来监听系统信号，用于优雅关闭
	go func() {
		// 创建一个通道来接收系统信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		
		// 等待信号
		sig := <-sigChan
		utils.Info("收到系统信号", map[string]string{"signal": sig.String()})
		
		// 停止工作池
		workerPool.Stop()
		utils.Info("工作池已停止", nil)
		
		// 退出程序
		os.Exit(0)
	}()
	
	// 初始化视频编辑服务
	editorService = service.NewVideoEditorService(taskQueue)
	utils.Info("视频编辑服务初始化完成", nil)
	
	// 创建Gin引擎
	r := gin.Default()
	utils.Info("Gin引擎创建完成", nil)
	
	// 定义健康检查端点
	r.GET("/health", func(c *gin.Context) {
		utils.Info("健康检查请求", map[string]string{"clientIP": c.ClientIP()})
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	
	// 提供静态文件服务
	r.StaticFile("/", "./web/index.html")
	r.Static("/static", "./web")
	
	// 定义API路由组
	apiGroup := r.Group("/api/v1")
	{
		apiGroup.POST("/video/edit", submitVideoEdit)
		apiGroup.GET("/video/edit/:taskId", getVideoEditStatus)
		apiGroup.DELETE("/video/edit/:taskId", cancelVideoEdit)
		
		// 添加WorkerPool管理接口
		apiGroup.GET("/workerpool/status", getWorkerPoolStatus)
		apiGroup.POST("/workerpool/resize", resizeWorkerPool)
		
		// 添加监控接口
		apiGroup.GET("/monitor/stats", monitorAPI.GetSystemStats)
		apiGroup.GET("/monitor/tasks/stats", monitorAPI.GetTaskStats)
		apiGroup.GET("/monitor/tasks", monitorAPI.GetTasks)
		apiGroup.GET("/monitor/tasks/:taskId", monitorAPI.GetTaskDetail)
		apiGroup.GET("/monitor/workers", monitorAPI.GetWorkerStats)
		// 添加任务管理接口
		apiGroup.POST("/monitor/tasks/retry", monitorAPI.RetryTask)
		apiGroup.POST("/monitor/tasks/cancel", monitorAPI.CancelTask)
		apiGroup.POST("/monitor/tasks/discard", monitorAPI.DiscardTask)
	}
	
	// 从环境变量获取端口，默认为8082
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	
	utils.Info("服务启动中", map[string]string{"port": port})
	log.Printf("Server starting on port %s", port)
	
	// 使用环境变量中的端口启动服务
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Error("服务启动失败", map[string]string{"error": err.Error()})
			log.Fatal("Failed to start server:", err)
		}
	}()
	
	// 等待中断信号以优雅地关闭服务器
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	utils.Info("服务正在关闭", nil)
	
	// 关闭工作池
	workerPool.Stop()
	utils.Info("工作池已停止", nil)
}

// submitVideoEdit 处理视频编辑任务提交请求
func submitVideoEdit(c *gin.Context) {
	utils.Info("收到视频编辑任务提交请求", map[string]string{"clientIP": c.ClientIP()})
	
	var req service.VideoEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warn("视频编辑请求格式错误", map[string]string{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	// 创建任务
	task, err := editorService.SubmitTask(&req)
	if err != nil {
		utils.Error("提交任务失败", map[string]string{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to submit task",
		})
		return
	}
	
	utils.Info("任务提交成功", map[string]string{"taskId": task.ID, "priority": string(rune(task.Priority + '0'))})
	
	c.JSON(http.StatusAccepted, gin.H{
		"taskId": task.ID,
		"status": task.Status,
		"priority": task.Priority, // 返回任务优先级
	})
}

// getVideoEditStatus 获取视频编辑任务状态
func getVideoEditStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	utils.Info("收到任务状态查询请求", map[string]string{"taskId": taskID, "clientIP": c.ClientIP()})
	
	task, err := editorService.GetTaskStatus(taskID)
	if err != nil {
		utils.Error("获取任务状态失败", map[string]string{"taskId": taskID, "error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task status",
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
	
	response := convertTaskToResponse(task)
	utils.Info("任务状态查询成功", map[string]string{"taskId": taskID, "status": task.Status})
	
	c.JSON(http.StatusOK, response)
}

// cancelVideoEdit 取消视频编辑任务
func cancelVideoEdit(c *gin.Context) {
	taskID := c.Param("taskId")
	utils.Info("收到任务取消请求", map[string]string{"taskId": taskID, "clientIP": c.ClientIP()})
	
	err := editorService.CancelTask(taskID)
	if err != nil {
		utils.Error("取消任务失败", map[string]string{"taskId": taskID, "error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to cancel task",
			"taskId": taskID,
		})
		return
	}
	
	utils.Info("任务取消成功", map[string]string{"taskId": taskID})
	c.JSON(http.StatusOK, gin.H{
		"message": "Task cancelled successfully",
		"taskId": taskID,
	})
}

// getWorkerPoolStatus 获取WorkerPool状态
func getWorkerPoolStatus(c *gin.Context) {
	utils.Info("收到WorkerPool状态查询请求", map[string]string{"clientIP": c.ClientIP()})
	
	status := gin.H{
		"workerCount": workerPool.GetWorkerCount(),
	}
	
	utils.Info("WorkerPool状态查询成功", map[string]string{"workerCount": strconv.Itoa(workerPool.GetWorkerCount())})
	c.JSON(http.StatusOK, status)
}

// resizeWorkerPool 调整WorkerPool大小
func resizeWorkerPool(c *gin.Context) {
	utils.Info("收到WorkerPool调整请求", map[string]string{"clientIP": c.ClientIP()})
	
	var req struct {
		Size int `json:"size"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Warn("WorkerPool调整请求格式错误", map[string]string{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	if req.Size <= 0 {
		utils.Warn("WorkerPool调整参数错误", map[string]string{"size": strconv.Itoa(req.Size)})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Size must be greater than 0",
		})
		return
	}
	
	workerPool.Resize(req.Size)
	utils.Info("WorkerPool调整完成", map[string]string{"newSize": strconv.Itoa(req.Size), "workerCount": strconv.Itoa(workerPool.GetWorkerCount())})
	
	c.JSON(http.StatusOK, gin.H{
		"message": "WorkerPool resized successfully",
		"workerCount": workerPool.GetWorkerCount(),
	})
}

// convertTaskToResponse 将任务转换为响应格式
func convertTaskToResponse(task *queue.Task) *TaskStatusResponse {
	response := &TaskStatusResponse{
		TaskID:   task.ID,
		Status:   task.Status,
		Progress: task.Progress,
		Message:  task.Error,
		Priority: task.Priority, // 添加优先级字段
	}
	
	if !task.Created.IsZero() {
		response.Created = task.Created.Format(time.RFC3339)
	}
	
	if !task.Started.IsZero() {
		response.Started = task.Started.Format(time.RFC3339)
	}
	
	if !task.Finished.IsZero() {
		response.Finished = task.Finished.Format(time.RFC3339)
	}
	
	if task.Result != "" {
		response.OutputURL = task.Result
	}
	
	return response
}